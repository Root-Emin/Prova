export type RendererMode = "development" | "production"

const DEVELOPMENT_ORIGIN = "http://127.0.0.1:3100"
const PRODUCTION_PROTOCOL = "app:"
const PRODUCTION_HOST = "prova"

export function isTrustedRendererUrl(rawUrl: string, mode: RendererMode): boolean {
  try {
    const url = new URL(rawUrl)

    if (mode === "development") {
      return url.origin === DEVELOPMENT_ORIGIN && !url.username && !url.password
    }

    return (
      url.protocol === PRODUCTION_PROTOCOL &&
      url.hostname === PRODUCTION_HOST &&
      url.port === "" &&
      !url.username &&
      !url.password
    )
  } catch {
    return false
  }
}
