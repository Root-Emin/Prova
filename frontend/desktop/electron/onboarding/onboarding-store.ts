import { promises as fs } from "node:fs"
import path from "node:path"
import { randomUUID } from "node:crypto"

import type { OnboardingState } from "../api-types"

type StoredOnboarding = {
  version: 1
  completedAt: string
}

/**
 * Whether the three intro screens have been seen. Not a secret, so it stays
 * out of ElectronSafeStorage: on Linux without a keyring that store fails
 * closed, and a missing keyring must not keep the window from opening.
 */
export class OnboardingStore {
  constructor(private readonly file: string) {}

  async getState(): Promise<OnboardingState> {
    let raw: string
    try {
      raw = await fs.readFile(this.file, "utf8")
    } catch (error) {
      if (isNodeError(error) && error.code === "ENOENT") return { completed: false }
      throw error
    }

    // A hand-edited or truncated file means "not seen yet", never a crash.
    try {
      const parsed: unknown = JSON.parse(raw)
      return { completed: isStoredOnboarding(parsed) }
    } catch {
      return { completed: false }
    }
  }

  async complete(): Promise<OnboardingState> {
    const stored: StoredOnboarding = { version: 1, completedAt: new Date().toISOString() }
    const temporaryFile = `${this.file}.${randomUUID()}.tmp`

    await fs.mkdir(path.dirname(this.file), { recursive: true })
    try {
      await fs.writeFile(temporaryFile, JSON.stringify(stored), { mode: 0o600 })
      await fs.rename(temporaryFile, this.file)
    } finally {
      await fs.unlink(temporaryFile).catch(() => undefined)
    }
    return { completed: true }
  }
}

function isStoredOnboarding(value: unknown): value is StoredOnboarding {
  if (typeof value !== "object" || value === null) return false
  const candidate = value as Partial<StoredOnboarding>
  return candidate.version === 1 && typeof candidate.completedAt === "string"
}

function isNodeError(error: unknown): error is NodeJS.ErrnoException {
  return error instanceof Error && "code" in error
}
