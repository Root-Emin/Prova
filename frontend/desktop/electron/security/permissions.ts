import type { Session, WebContents } from "electron"

import type { RendererMode } from "./trusted-renderer"
import { isTrustedRendererUrl } from "./trusted-renderer"

export function installPermissionPolicy(targetSession: Session, mode: RendererMode): void {
  targetSession.setPermissionCheckHandler((webContents, permission, requestingOrigin, details) => {
    if (permission !== "media" || details.mediaType !== "audio" || !details.isMainFrame) return false
    return isTrustedPermissionRequester(webContents, [
      requestingOrigin,
      details.requestingUrl,
      details.securityOrigin,
      details.embeddingOrigin,
    ], mode)
  })

  targetSession.setPermissionRequestHandler((webContents, permission, callback, details) => {
    if (
      permission !== "media" ||
      !("mediaTypes" in details) ||
      !details.isMainFrame
    ) {
      callback(false)
      return
    }

    const mediaTypes = details.mediaTypes ?? []
    const audioOnly = mediaTypes.length === 1 && mediaTypes[0] === "audio"
    callback(audioOnly && isTrustedPermissionRequester(webContents, [
      details.requestingUrl,
      details.securityOrigin,
    ], mode))
  })
}

function isTrustedPermissionRequester(
  webContents: WebContents | null,
  requestingUrls: Array<string | undefined>,
  mode: RendererMode,
): boolean {
  if (!webContents || webContents.isDestroyed()) return false
  const presentUrls = requestingUrls.filter((url): url is string => url !== undefined)
  return (
    presentUrls.length > 0 &&
    presentUrls.every((url) => isTrustedRendererUrl(url, mode)) &&
    isTrustedRendererUrl(webContents.getURL(), mode)
  )
}
