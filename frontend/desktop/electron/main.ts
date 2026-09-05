import path from "node:path"
import { createPublicKey, verify } from "node:crypto"

import { app, BrowserWindow, Menu, session } from "electron"

import { DeviceIdentityService } from "./device/device-identity"
import { DesktopAuthService } from "./auth/auth-service"
import { registerIpcHandlers } from "./ipc/register-handlers"
import { OnboardingStore } from "./onboarding/onboarding-store"
import { APP_URL, installAppProtocol, registerAppScheme } from "./protocol"
import { installPermissionPolicy } from "./security/permissions"
import { ElectronSafeStorage } from "./security/secure-storage"
import type { RendererMode } from "./security/trusted-renderer"
import { isTrustedRendererUrl } from "./security/trusted-renderer"

const DEVELOPMENT_URL = "http://127.0.0.1:3100"
const smokeProductionRenderer = process.env.PROVA_ELECTRON_SMOKE_TEST === "1"
const mode: RendererMode = app.isPackaged || smokeProductionRenderer ? "production" : "development"

registerAppScheme()

if (!app.requestSingleInstanceLock()) {
  app.quit()
} else {
  void app.whenReady().then(startApplication).catch((error) => {
    console.error("Failed to start Prova desktop", error)
    app.exit(1)
  })
}

async function startApplication(): Promise<void> {
  const exportRoot = path.join(app.getAppPath(), "out")
  await installAppProtocol(exportRoot)
  installPermissionPolicy(session.defaultSession, mode)
  installProductionHeaders()

  const storage = new ElectronSafeStorage(path.join(app.getPath("userData"), "secure-storage"))
  const deviceIdentity = new DeviceIdentityService(storage)
  const authService = new DesktopAuthService(deviceIdentity, storage)
  const onboarding = new OnboardingStore(path.join(app.getPath("userData"), "onboarding.json"))
  const trustedWebContents = new Set<number>()
  registerIpcHandlers(mode, deviceIdentity, authService, onboarding, trustedWebContents)

  Menu.setApplicationMenu(null)
  // The three intro screens come before login, and only on a first launch.
  const { completed } = await onboarding.getState()
  const window = createMainWindow(trustedWebContents, completed ? "login" : "onboarding")

  if (process.platform === "darwin") {
    app.on("activate", () => {
      if (BrowserWindow.getAllWindows().length > 0) return
      void onboarding.getState().then((state) => {
        createMainWindow(trustedWebContents, state.completed ? "login" : "onboarding")
      })
    })
  }

  if (smokeProductionRenderer) {
    window.webContents.once("did-finish-load", async () => {
      try {
        const result = await window.webContents.executeJavaScript(`(async () => ({
          ...(await (async () => {
            const challenge = "prova-smoke-challenge-0123456789"
            const registration = await window.prova.device.getRegistrationInfo()
            const routeResponse = await fetch("/session/")
            return {
              challenge,
              fingerprintLength: registration.fingerprint.length,
              routeOk: routeResponse.ok && (await routeResponse.text()).includes("<!DOCTYPE html>"),
            }
          })()),
          title: document.title,
          styleSheetCount: document.styleSheets.length,
          systemInfo: await window.prova.system.getInfo()
        }))()`)
        const signature = await deviceIdentity.signChallenge(result.challenge)
        const registration = await deviceIdentity.getRegistrationInfo()
        const publicKey = createPublicKey({
          key: { kty: "OKP", crv: "Ed25519", x: Buffer.from(registration.publicKey, "base64").toString("base64url") },
          format: "jwk",
        })
        const signatureValid = verify(
          null,
          Buffer.from(result.challenge, "utf8"),
          publicKey,
          Buffer.from(signature, "base64"),
        )
        if (
          result.title !== "Prova" || result.styleSheetCount < 1 ||
          result.systemInfo.platform === undefined || !signatureValid ||
          result.fingerprintLength !== 64 || !result.routeOk
        ) {
          throw new Error(`Unexpected renderer state: ${JSON.stringify(result)}`)
        }
        console.log(`PROVA_ELECTRON_SMOKE_OK ${window.webContents.getURL()}`)
        finishSmoke(0)
      } catch (error) {
        console.error("PROVA_ELECTRON_SMOKE_FAILED", error)
        finishSmoke(1)
      }
    })
    window.webContents.once("did-fail-load", (_event, code, description) => {
      console.error(`PROVA_ELECTRON_SMOKE_FAILED ${code} ${description}`)
      finishSmoke(1)
    })
  }
}

function finishSmoke(exitCode: number): void {
  // This path is enabled only by PROVA_ELECTRON_SMOKE_TEST. A direct exit is
  // intentional so CI does not remain attached to macOS helper processes.
  process.exit(exitCode)
}

function createMainWindow(
  trustedWebContents: Set<number>,
  entryRoute: "login" | "onboarding",
): BrowserWindow {
  const window = new BrowserWindow({
    width: 1180,
    height: 780,
    minWidth: 900,
    minHeight: 640,
    show: !smokeProductionRenderer,
    webPreferences: {
      preload: path.join(__dirname, "preload.js"),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      webSecurity: true,
      allowRunningInsecureContent: false,
      webviewTag: false,
      devTools: mode === "development",
      spellcheck: false,
    },
  })

  window.webContents.setWindowOpenHandler(() => ({ action: "deny" }))
  trustedWebContents.add(window.webContents.id)
  window.on("closed", () => trustedWebContents.delete(window.webContents.id))
  const preventUntrustedNavigation = (event: Electron.Event, url: string) => {
    if (!isTrustedRendererUrl(url, mode)) event.preventDefault()
  }
  window.webContents.on("will-navigate", preventUntrustedNavigation)
  window.webContents.on("will-redirect", preventUntrustedNavigation)
  window.webContents.on("will-attach-webview", (event) => event.preventDefault())

  if (!smokeProductionRenderer) {
    window.once("ready-to-show", () => window.show())
  }
  const initialRoute = mode === "production"
    ? `${APP_URL}${entryRoute}/`
    : `${DEVELOPMENT_URL}/${entryRoute}`
  void window.loadURL(initialRoute).catch((error) => {
    console.error("Failed to load Prova renderer", error)
    if (smokeProductionRenderer) finishSmoke(1)
  })
  return window
}

function installProductionHeaders(): void {
  session.defaultSession.webRequest.onHeadersReceived((details, callback) => {
    if (!isTrustedRendererUrl(details.url, "production")) {
      callback({ responseHeaders: details.responseHeaders })
      return
    }

    callback({
      responseHeaders: {
        ...details.responseHeaders,
        "Content-Security-Policy": [
          "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self'; connect-src 'self'; media-src 'self' blob:; object-src 'none'; frame-src 'none'; base-uri 'self'; form-action 'self'",
        ],
        "X-Content-Type-Options": ["nosniff"],
      },
    })
  })
}

app.on("window-all-closed", () => {
  if (process.platform !== "darwin") app.quit()
})

app.on("second-instance", () => {
  const window = BrowserWindow.getAllWindows()[0]
  if (!window) return
  if (window.isMinimized()) window.restore()
  window.focus()
})
