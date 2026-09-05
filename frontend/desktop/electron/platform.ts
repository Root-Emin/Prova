import type { NormalizedPlatform } from "./api-types"

export function normalizePlatform(platform: NodeJS.Platform): NormalizedPlatform {
  switch (platform) {
    case "darwin":
      return "macos"
    case "win32":
      return "windows"
    case "linux":
      return "linux"
    default:
      throw new Error(`Unsupported desktop platform: ${platform}`)
  }
}
