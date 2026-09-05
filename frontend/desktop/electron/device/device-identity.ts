import {
  createHash,
  createPrivateKey,
  createPublicKey,
  generateKeyPairSync,
  randomUUID,
  sign,
} from "node:crypto"
import os from "node:os"

import { app } from "electron"

import type { DeviceInfo, DeviceRegistrationInfo } from "../api-types"
import { normalizePlatform } from "../platform"
import type { SecureStorage } from "../security/secure-storage"

type StoredDeviceIdentity = {
  version: 1
  deviceId: string
  publicKey: string
  privateKeyPkcs8: string
}

const STORAGE_KEY = "device-identity-v1"
const MAX_CHALLENGE_LENGTH = 512

export class DeviceIdentityService {
  private identityPromise: Promise<StoredDeviceIdentity> | null = null

  constructor(private readonly storage: SecureStorage) {}

  getInfo(): DeviceInfo {
    return {
      displayName: os.hostname() || "Prova Desktop",
      platform: normalizePlatform(process.platform),
      osRelease: os.release(),
      appVersion: app.getVersion(),
    }
  }

  async getRegistrationInfo(): Promise<DeviceRegistrationInfo> {
    const identity = await this.loadOrCreate()
    return {
      ...this.getInfo(),
      fingerprint: fingerprintOf(identity),
      publicKey: identity.publicKey,
    }
  }

  async signChallenge(challenge: string): Promise<string> {
    const challengeBytes = typeof challenge === "string" ? Buffer.from(challenge, "utf8") : null
    if (!challengeBytes || challengeBytes.length < 16 || challengeBytes.length > MAX_CHALLENGE_LENGTH) {
      throw new Error("Invalid device challenge")
    }

    const identity = await this.loadOrCreate()
    const privateKey = createPrivateKey({
      key: Buffer.from(identity.privateKeyPkcs8, "base64"),
      format: "der",
      type: "pkcs8",
    })
    return sign(null, challengeBytes, privateKey).toString("base64")
  }

  private async loadOrCreate(): Promise<StoredDeviceIdentity> {
    this.identityPromise ??= this.readOrCreate()
    return this.identityPromise
  }

  private async readOrCreate(): Promise<StoredDeviceIdentity> {
    const existing = await this.storage.get(STORAGE_KEY)
    if (existing) {
      return parseStoredDeviceIdentity(existing)
    }

    const { publicKey, privateKey } = generateKeyPairSync("ed25519")
    const publicJwk = publicKey.export({ format: "jwk" })
    if (!publicJwk.x) throw new Error("Unable to export Ed25519 public key")

    const identity: StoredDeviceIdentity = {
      version: 1,
      deviceId: randomUUID(),
      publicKey: Buffer.from(publicJwk.x, "base64url").toString("base64"),
      privateKeyPkcs8: privateKey.export({ format: "der", type: "pkcs8" }).toString("base64"),
    }
    await this.storage.set(STORAGE_KEY, JSON.stringify(identity))
    return identity
  }
}

function fingerprintOf(identity: StoredDeviceIdentity): string {
  return createHash("sha256")
    .update(`${identity.deviceId}:${identity.publicKey}`, "utf8")
    .digest("hex")
}

export function parseStoredDeviceIdentity(raw: string): StoredDeviceIdentity {
  let parsed: unknown
  try {
    parsed = JSON.parse(raw)
  } catch {
    throw new Error("Stored device identity is invalid")
  }
  if (
    typeof parsed !== "object" || parsed === null ||
    !("version" in parsed) || parsed.version !== 1 ||
    !("deviceId" in parsed) || typeof parsed.deviceId !== "string" ||
    !("publicKey" in parsed) || typeof parsed.publicKey !== "string" ||
    !("privateKeyPkcs8" in parsed) || typeof parsed.privateKeyPkcs8 !== "string"
  ) {
    throw new Error("Stored device identity is invalid")
  }

  const identity = parsed as StoredDeviceIdentity
  if (!/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(identity.deviceId)) {
    throw new Error("Stored device identity is invalid")
  }

  const publicKey = decodeCanonicalBase64(identity.publicKey, 32)
  const privateKeyDer = decodeCanonicalBase64(identity.privateKeyPkcs8, 48)
  try {
    const privateKey = createPrivateKey({ key: privateKeyDer, format: "der", type: "pkcs8" })
    const privateJwk = createPublicKey(privateKey).export({ format: "jwk" })
    if (privateJwk.kty !== "OKP" || privateJwk.crv !== "Ed25519" || !privateJwk.x) {
      throw new Error("not Ed25519")
    }
    const derivedPublicKey = Buffer.from(privateJwk.x, "base64url")
    if (!derivedPublicKey.equals(publicKey)) throw new Error("key mismatch")
  } catch {
    throw new Error("Stored device identity is invalid")
  }

  return identity
}

function decodeCanonicalBase64(value: string, expectedLength: number): Buffer {
  if (!/^[A-Za-z0-9+/]+={0,2}$/.test(value) || value.length % 4 !== 0) {
    throw new Error("Stored device identity is invalid")
  }
  const decoded = Buffer.from(value, "base64")
  if (decoded.length !== expectedLength || decoded.toString("base64") !== value) {
    throw new Error("Stored device identity is invalid")
  }
  return decoded
}
