import { promises as fs } from "node:fs"
import path from "node:path"
import { randomUUID } from "node:crypto"

import { safeStorage } from "electron"

export type SecureStorageStatus = {
  available: boolean
  backend: string
  reason?: string
}

export interface SecureStorage {
  status(): Promise<SecureStorageStatus>
  get(key: string): Promise<string | null>
  set(key: string, value: string): Promise<void>
  delete(key: string): Promise<void>
}

export interface SafeStorageAdapter {
  isAsyncEncryptionAvailable(): Promise<boolean>
  getSelectedStorageBackend(): string
  encryptStringAsync(value: string): Promise<Buffer>
  decryptStringAsync(value: Buffer): Promise<{ result: string; shouldReEncrypt: boolean }>
}

const VALID_KEY = /^[a-z0-9][a-z0-9._-]{0,63}$/

export class ElectronSafeStorage implements SecureStorage {
  private readonly adapter: SafeStorageAdapter

  constructor(
    private readonly directory: string,
    adapter: SafeStorageAdapter = safeStorage,
  ) {
    this.adapter = adapter
  }

  async status(): Promise<SecureStorageStatus> {
    const available = await this.adapter.isAsyncEncryptionAvailable()
    const backend = process.platform === "linux"
      ? this.adapter.getSelectedStorageBackend()
      : process.platform === "darwin"
        ? "keychain"
        : "dpapi"
    return storageAvailability(process.platform, available, backend)
  }

  async get(key: string): Promise<string | null> {
    await this.assertAvailable()
    const file = this.pathFor(key)
    let encrypted: Buffer

    try {
      encrypted = await fs.readFile(file)
    } catch (error) {
      if (isNodeError(error) && error.code === "ENOENT") return null
      throw error
    }

    let decrypted: { result: string; shouldReEncrypt: boolean }
    try {
      decrypted = await this.adapter.decryptStringAsync(encrypted)
    } catch {
      throw new Error("Secure storage data is corrupt or cannot be decrypted")
    }
    if (decrypted.shouldReEncrypt) {
      await this.set(key, decrypted.result)
    }
    return decrypted.result
  }

  async set(key: string, value: string): Promise<void> {
    await this.assertAvailable()
    const file = this.pathFor(key)
    const temporaryFile = `${file}.${randomUUID()}.tmp`
    const encrypted = await this.adapter.encryptStringAsync(value)

    await this.ensureDirectory()
    try {
      await fs.writeFile(temporaryFile, encrypted, { mode: 0o600 })
      if (process.platform !== "win32") await fs.chmod(temporaryFile, 0o600)
      await fs.rename(temporaryFile, file)
      if (process.platform !== "win32") await fs.chmod(file, 0o600)
    } finally {
      await fs.unlink(temporaryFile).catch(() => undefined)
    }
  }

  async delete(key: string): Promise<void> {
    try {
      await fs.unlink(this.pathFor(key))
    } catch (error) {
      if (!isNodeError(error) || error.code !== "ENOENT") throw error
    }
  }

  private pathFor(key: string): string {
    if (!VALID_KEY.test(key)) throw new Error("Invalid secure-storage key")
    return path.join(this.directory, `${key}.bin`)
  }

  private async assertAvailable(): Promise<void> {
    const current = await this.status()
    if (!current.available) {
      throw new Error(`Secure storage unavailable (${current.backend}): ${current.reason}`)
    }
  }

  private async ensureDirectory(): Promise<void> {
    await fs.mkdir(this.directory, { recursive: true, mode: 0o700 })
    const directoryStat = await fs.lstat(this.directory)
    if (!directoryStat.isDirectory() || directoryStat.isSymbolicLink()) {
      throw new Error("Secure storage directory is invalid")
    }
    if (process.platform !== "win32") await fs.chmod(this.directory, 0o700)
  }
}

export function storageAvailability(
  platform: NodeJS.Platform,
  asyncEncryptionAvailable: boolean,
  backend: string,
): SecureStorageStatus {
  if (!asyncEncryptionAvailable) {
    return { available: false, backend, reason: "OS encryption is unavailable" }
  }

  // Electron's basic_text Linux backend uses a hard-coded password. Persisting
  // device private keys or refresh tokens with it would only obfuscate plaintext.
  if (
    platform === "linux" &&
    !new Set(["gnome_libsecret", "kwallet", "kwallet5", "kwallet6"]).has(backend)
  ) {
    return {
      available: false,
      backend,
      reason: "A supported Linux secret service is required",
    }
  }

  return { available: true, backend }
}

function isNodeError(error: unknown): error is NodeJS.ErrnoException {
  return error instanceof Error && "code" in error
}
