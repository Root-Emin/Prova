const assert = require("node:assert/strict")
const crypto = require("node:crypto")
const fs = require("node:fs/promises")
const os = require("node:os")
const path = require("node:path")

const { ElectronSafeStorage, storageAvailability } = require("../dist-electron/security/secure-storage.js")
const { DesktopAuthService } = require("../dist-electron/auth/auth-service.js")
const { installPermissionPolicy } = require("../dist-electron/security/permissions.js")
const { isTrustedRendererUrl } = require("../dist-electron/security/trusted-renderer.js")
const { parseStoredDeviceIdentity } = require("../dist-electron/device/device-identity.js")
const { OnboardingStore } = require("../dist-electron/onboarding/onboarding-store.js")

async function main() {
  const preloadSource = await fs.readFile(path.join(__dirname, "..", "electron", "preload.ts"), "utf8")
  assert.equal(preloadSource.includes("signChallenge"), false)
  assert.equal(preloadSource.includes("deviceSignChallenge"), false)

  assert.equal(isTrustedRendererUrl("app://prova/", "production"), true)
  for (const url of [
    "app://evil/",
    "app://user:pass@prova/",
    "app://prova:443/",
    "https://prova/",
    "app://prova.evil/",
  ]) assert.equal(isTrustedRendererUrl(url, "production"), false, url)
  assert.equal(isTrustedRendererUrl("http://127.0.0.1:3100/", "development"), true)
  assert.equal(isTrustedRendererUrl("http://user:pass@127.0.0.1:3100/", "development"), false)

  assert.equal(storageAvailability("linux", true, "basic_text").available, false)
  assert.equal(storageAvailability("linux", true, "unknown").available, false)
  assert.equal(storageAvailability("linux", true, "gnome_libsecret").available, true)
  assert.equal(storageAvailability("darwin", false, "keychain").available, false)

  const authCalls = []
  const previousFetch = globalThis.fetch
  globalThis.fetch = async (_url, init) => {
    const body = JSON.parse(init.body)
    authCalls.push(body.query)
    if (body.query.includes("RequestLoginCode")) {
      return { ok: true, json: async () => ({ data: {
        requestLoginCode: { sent: true, expiresInSeconds: 600, resendAfterSeconds: 60 },
      } }) }
    }
    if (body.query.includes("RequestDeviceChallenge")) {
      return { ok: true, json: async () => ({ data: {
        requestDeviceChallenge: { challenge: "challenge-from-server-123456" },
      } }) }
    }
    return { ok: true, json: async () => ({ data: {
      verifyLoginCode: {
        accessToken: "access-token",
        refreshToken: "refresh-token",
        expiresAt: "2030-01-01T00:00:00Z",
        organizationId: "org-id",
        user: { email: "person@example.com" },
        device: { isNew: true },
      },
    } }) }
  }
  try {
    let storedSession = null
    const authStorage = {
      status: async () => ({ available: true, backend: "test" }),
      get: async () => null,
      set: async (_key, value) => { storedSession = JSON.parse(value) },
      delete: async () => undefined,
    }
    const authIdentity = {
      getRegistrationInfo: async () => ({
        displayName: "Test device",
        platform: "macos",
        osRelease: "test",
        appVersion: "test",
        fingerprint: "a".repeat(64),
        publicKey: "A".repeat(44),
      }),
      signChallenge: async (challenge) => `signature:${challenge}`,
    }
    assert.throws(
      () => new DesktopAuthService(authIdentity, authStorage, "http://example.com/graphql"),
      /HTTPS/,
    )
    const auth = new DesktopAuthService(authIdentity, authStorage, "http://127.0.0.1:8080/graphql")
    assert.deepEqual(await auth.requestLoginCode(" Person@Example.com "), {
      sent: true, expiresInSeconds: 600, resendAfterSeconds: 60,
    })
    assert.deepEqual(await auth.verifyLoginCode("person@example.com", "123456"), {
      deviceIsNew: true, expiresAt: "2030-01-01T00:00:00Z",
    })
    assert.equal(storedSession.email, "person@example.com")
    assert.equal(authCalls.length, 3)
  } finally {
    globalThis.fetch = previousFetch
  }

  const { publicKey, privateKey } = crypto.generateKeyPairSync("ed25519")
  const publicJwk = publicKey.export({ format: "jwk" })
  const validIdentity = {
    version: 1,
    deviceId: "123e4567-e89b-42d3-a456-426614174000",
    publicKey: Buffer.from(publicJwk.x, "base64url").toString("base64"),
    privateKeyPkcs8: privateKey.export({ format: "der", type: "pkcs8" }).toString("base64"),
  }
  assert.deepEqual(parseStoredDeviceIdentity(JSON.stringify(validIdentity)), validIdentity)
  assert.throws(() => parseStoredDeviceIdentity(JSON.stringify({ ...validIdentity, publicKey: "AAAA" })), /invalid/)
  assert.throws(() => parseStoredDeviceIdentity(JSON.stringify({
    ...validIdentity,
    privateKeyPkcs8: Buffer.alloc(48).toString("base64"),
  })), /invalid/)

  const root = await fs.mkdtemp(path.join(os.tmpdir(), "prova-secure-storage-"))
  try {
    let available = false
    const adapter = {
      async isAsyncEncryptionAvailable() { return available },
      getSelectedStorageBackend() { return "test" },
      async encryptStringAsync(value) { return Buffer.from(`encrypted:${value}`) },
      async decryptStringAsync(value) {
        const text = value.toString()
        if (!text.startsWith("encrypted:")) throw new Error("corrupt")
        return { result: text.slice("encrypted:".length), shouldReEncrypt: false }
      },
    }
    const storage = new ElectronSafeStorage(root, adapter)
    await assert.rejects(storage.get("missing"), /Secure storage unavailable/)
    available = true
    assert.equal(await storage.get("missing"), null)
    await storage.set("token", "secret")
    assert.equal(await storage.get("token"), "secret")
    assert.throws(() => storage.pathFor?.("../escape"))
    const files = await fs.readdir(root)
    assert.deepEqual(files, ["token.bin"])
    if (process.platform !== "win32") {
      assert.equal((await fs.stat(root)).mode & 0o777, 0o700)
      assert.equal((await fs.stat(path.join(root, "token.bin"))).mode & 0o777, 0o600)
    }
    await fs.writeFile(path.join(root, "broken.bin"), "not-encrypted")
    await assert.rejects(storage.get("broken"), /corrupt or cannot be decrypted/)
    if (process.platform !== "win32") {
      const symlinkRoot = `${root}-link`
      await fs.symlink(root, symlinkRoot)
      await assert.rejects(
        new ElectronSafeStorage(symlinkRoot, adapter).set("linked", "value"),
        /directory is invalid/,
      )
      await fs.unlink(symlinkRoot)
    }
    available = false
    await assert.rejects(storage.set("later", "value"), /Secure storage unavailable/)
  } finally {
    await fs.rm(root, { recursive: true, force: true })
  }

  const onboardingRoot = await fs.mkdtemp(path.join(os.tmpdir(), "prova-onboarding-"))
  try {
    const file = path.join(onboardingRoot, "onboarding.json")
    const onboarding = new OnboardingStore(file)
    assert.deepEqual(await onboarding.getState(), { completed: false })
    assert.deepEqual(await onboarding.complete(), { completed: true })
    assert.deepEqual(await onboarding.getState(), { completed: true })
    // A hand-edited or truncated file sends the user through the intro again
    // instead of keeping the window from opening.
    await fs.writeFile(file, "{not json")
    assert.deepEqual(await onboarding.getState(), { completed: false })
    await fs.writeFile(file, JSON.stringify({ version: 2 }))
    assert.deepEqual(await onboarding.getState(), { completed: false })
  } finally {
    await fs.rm(onboardingRoot, { recursive: true, force: true })
  }

  let checkHandler
  let requestHandler
  installPermissionPolicy({
    setPermissionCheckHandler(handler) { checkHandler = handler },
    setPermissionRequestHandler(handler) { requestHandler = handler },
  }, "production")
  const webContents = {
    isDestroyed: () => false,
    getURL: () => "app://prova/",
  }
  assert.equal(checkHandler(webContents, "media", "app://prova/", {
    mediaType: "audio", isMainFrame: true, requestingUrl: "app://prova/",
  }), true)
  assert.equal(checkHandler(webContents, "media", "app://prova/", {
    mediaType: "audio", isMainFrame: false, requestingUrl: "app://prova/",
  }), false)
  assert.equal(checkHandler(webContents, "media", "app://user:pass@prova/", {
    mediaType: "audio", isMainFrame: true, requestingUrl: "app://prova/",
  }), false)
  const request = (details) => new Promise((resolve) => requestHandler(webContents, "media", resolve, details))
  assert.equal(await request({ isMainFrame: true, requestingUrl: "app://prova/", mediaTypes: ["audio"] }), true)
  assert.equal(await request({ isMainFrame: true, requestingUrl: "app://prova/", mediaTypes: ["audio", "video"] }), false)
  assert.equal(await request({ isMainFrame: false, requestingUrl: "app://prova/", mediaTypes: ["audio"] }), false)
  assert.equal(await request({ isMainFrame: true, requestingUrl: "app://evil/", mediaTypes: ["audio"] }), false)

  console.log("Security policy and secure-storage checks passed")
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
