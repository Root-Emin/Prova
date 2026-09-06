import { createHmac, timingSafeEqual } from "node:crypto"

export const sessionCookie = "prova_admin_session"
export const sessionLifetimeSeconds = 60 * 60 * 8

export type Session = {
  email: string
  name: string
  role: "admin"
  expiresAt: number
}

function secret(): string {
  return process.env.AUTH_SESSION_SECRET ?? "prova-local-session-secret"
}

function encode(value: string): string {
  return Buffer.from(value, "utf8").toString("base64url")
}

function sign(value: string): string {
  return createHmac("sha256", secret()).update(value).digest("base64url")
}

export function makeSession(email: string): string {
  const payload: Session = {
    email,
    name: "Demo Yönetici",
    role: "admin",
    expiresAt: Date.now() + sessionLifetimeSeconds * 1000,
  }
  const encoded = encode(JSON.stringify(payload))
  return `${encoded}.${sign(encoded)}`
}

export function readSession(value: string | undefined): Session | null {
  if (!value) return null
  const [encoded, signature] = value.split(".")
  if (!encoded || !signature) return null

  const expected = sign(encoded)
  const actualBuffer = Buffer.from(signature)
  const expectedBuffer = Buffer.from(expected)
  if (
    actualBuffer.length !== expectedBuffer.length ||
    !timingSafeEqual(actualBuffer, expectedBuffer)
  ) {
    return null
  }

  try {
    const payload = JSON.parse(
      Buffer.from(encoded, "base64url").toString("utf8"),
    ) as Session
    if (
      payload.role !== "admin" ||
      typeof payload.email !== "string" ||
      typeof payload.expiresAt !== "number" ||
      payload.expiresAt <= Date.now()
    ) {
      return null
    }
    return payload
  } catch {
    return null
  }
}

