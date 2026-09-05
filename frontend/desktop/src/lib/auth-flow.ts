import type { LoginCodeResponse } from "../../electron/api-types"

type PendingLogin = {
  email: string
  expiresInSeconds: number
  resendAfterSeconds: number
}

let pendingLogin: PendingLogin | null = null

export function setPendingLogin(email: string, response: LoginCodeResponse): void {
  pendingLogin = { email, ...response }
}

export function getPendingLogin(): PendingLogin | null {
  return pendingLogin
}

export function clearPendingLogin(): void {
  pendingLogin = null
}
