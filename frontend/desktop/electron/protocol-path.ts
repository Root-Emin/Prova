import { promises as fs } from "node:fs"
import path from "node:path"

const APP_PROTOCOL = "app:"
const APP_HOST = "prova"

/** Maps an app://prova request to one file inside the Next.js export root. */
export function resolveAppRequest(exportRoot: string, requestUrl: string): string | null {
  const rawPath = extractRawPath(requestUrl)
  if (rawPath === null || hasUnsafePathEncoding(rawPath)) return null

  let url: URL
  try {
    url = new URL(requestUrl)
  } catch {
    return null
  }

  if (url.protocol !== APP_PROTOCOL || url.hostname !== APP_HOST || url.username || url.password || url.port) {
    return null
  }

  let pathname: string
  try {
    pathname = decodeURIComponent(url.pathname)
  } catch {
    return null
  }

  if (pathname.includes("\0") || pathname.includes("\\")) return null

  const segments = pathname.split("/")
  if (segments.some((segment) => segment === "." || segment === ".." || segment.includes(":"))) return null

  let relativePath = segments.filter(Boolean).join("/")
  if (relativePath === "" || pathname.endsWith("/")) {
    relativePath = path.posix.join(relativePath, "index.html")
  } else if (path.posix.extname(relativePath) === "") {
    relativePath = path.posix.join(relativePath, "index.html")
  }

  const root = path.resolve(exportRoot)
  const candidate = path.resolve(root, ...relativePath.split("/"))
  if (!isPathInside(root, candidate)) return null
  return candidate
}

/**
 * Resolves symlinks before a file is served. The syntactic check above is not
 * sufficient if an export directory is modified to contain a link outside it.
 */
export async function resolveExistingFileWithinRoot(
  exportRoot: string,
  candidate: string,
): Promise<string | null> {
  try {
    const [realRoot, realCandidate] = await Promise.all([
      fs.realpath(exportRoot),
      fs.realpath(candidate),
    ])
    if (!isPathInside(realRoot, realCandidate)) return null
    return (await fs.stat(realCandidate)).isFile() ? realCandidate : null
  } catch {
    return null
  }
}

type PathOperations = Pick<typeof path, "relative" | "isAbsolute" | "sep">

export function isPathInside(
  root: string,
  candidate: string,
  pathOperations: PathOperations = path,
): boolean {
  const relative = pathOperations.relative(root, candidate)
  return (
    relative !== "" &&
    relative !== ".." &&
    !relative.startsWith(`..${pathOperations.sep}`) &&
    !pathOperations.isAbsolute(relative)
  )
}

function extractRawPath(requestUrl: string): string | null {
  const schemeEnd = requestUrl.indexOf("://")
  if (schemeEnd < 0) return null

  const authorityStart = schemeEnd + 3
  const relativeAuthorityEnd = requestUrl.slice(authorityStart).search(/[/?#]/)
  const authorityEnd = relativeAuthorityEnd < 0 ? -1 : authorityStart + relativeAuthorityEnd
  if (authorityEnd < 0 || requestUrl[authorityEnd] !== "/") return "/"

  const suffix = requestUrl.slice(authorityEnd)
  const pathEnd = suffix.search(/[?#]/)
  return pathEnd < 0 ? suffix : suffix.slice(0, pathEnd)
}

function hasUnsafePathEncoding(rawPath: string): boolean {
  let current = rawPath

  // Decode more than Chromium does so double-encoded traversal cannot become
  // dangerous if URL handling behavior changes in a future Electron release.
  for (let depth = 0; depth < 3; depth += 1) {
    if (current.includes("\0") || current.includes("\\")) return true
    const segments = current.split("/")
    if (segments.some((segment) => segment === "." || segment === ".." || segment.includes(":"))) {
      return true
    }

    let decoded: string
    try {
      decoded = decodeURIComponent(current)
    } catch {
      return true
    }
    if (decoded === current) return false
    current = decoded
  }

  return current !== rawPath && hasEncodedPathControl(current)
}

function hasEncodedPathControl(pathname: string): boolean {
  return /%(?:00|2e|2f|3a|5c)/i.test(pathname)
}
