import os from "node:os"

import {
  app,
  BrowserWindow,
  ipcMain,
  systemPreferences,
  type IpcMainInvokeEvent,
} from "electron"

import type { MicrophonePermissionStatus, SystemInfo } from "../api-types"
import type { DesktopAuthService } from "../auth/auth-service"
import type { DeviceIdentityService } from "../device/device-identity"
import type { OnboardingStore } from "../onboarding/onboarding-store"
import { normalizePlatform } from "../platform"
import type { RendererMode } from "../security/trusted-renderer"
import { isTrustedRendererUrl } from "../security/trusted-renderer"
import { IPC_CHANNELS } from "./channels"

export function registerIpcHandlers(
  mode: RendererMode,
  deviceIdentity: DeviceIdentityService,
  authService: DesktopAuthService,
  onboarding: OnboardingStore,
  trustedWebContents: ReadonlySet<number>,
): void {
  const handle = <TArgs extends unknown[], TResult>(
    channel: string,
    handler: (...args: TArgs) => TResult | Promise<TResult>,
  ) => {
    ipcMain.handle(channel, (event, ...args: TArgs) => {
      assertTrustedSender(event, mode, trustedWebContents)
      return handler(...args)
    })
  }

  handle(IPC_CHANNELS.systemGetInfo, (): SystemInfo => ({
    platform: normalizePlatform(process.platform),
    osRelease: os.release(),
    appVersion: app.getVersion(),
  }))
  handle(IPC_CHANNELS.deviceGetInfo, () => deviceIdentity.getInfo())
  handle(IPC_CHANNELS.deviceGetRegistrationInfo, () => deviceIdentity.getRegistrationInfo())
  handle(IPC_CHANNELS.permissionsGetMicrophoneStatus, getMicrophoneStatus)
  handle(IPC_CHANNELS.authRequestLoginCode, (email: string) => authService.requestLoginCode(email))
  handle(IPC_CHANNELS.authLogin, (email: string, password: string) => authService.login(email, password))
  handle(IPC_CHANNELS.authVerifyLoginCode, (email: string, code: string) => authService.verifyLoginCode(email, code))
  handle(IPC_CHANNELS.onboardingGetState, () => onboarding.getState())
  handle(IPC_CHANNELS.onboardingComplete, () => onboarding.complete())
}

function assertTrustedSender(
  event: IpcMainInvokeEvent,
  mode: RendererMode,
  trustedWebContents: ReadonlySet<number>,
): void {
  const owner = BrowserWindow.fromWebContents(event.sender)
  const frame = event.senderFrame
  if (
    !owner || owner.isDestroyed() ||
    !trustedWebContents.has(event.sender.id) ||
    !frame || frame !== event.sender.mainFrame ||
    !isTrustedRendererUrl(frame.url, mode)
  ) {
    throw new Error("Untrusted IPC sender")
  }
}

function getMicrophoneStatus(): MicrophonePermissionStatus {
  if (process.platform !== "darwin" && process.platform !== "win32") return "unknown"

  const status = systemPreferences.getMediaAccessStatus("microphone")
  switch (status) {
    case "granted":
    case "denied":
    case "restricted":
    case "not-determined":
      return status
    default:
      return "unknown"
  }
}
