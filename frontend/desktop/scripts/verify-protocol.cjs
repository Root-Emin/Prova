const assert = require("node:assert/strict")
const fs = require("node:fs")
const os = require("node:os")
const path = require("node:path")

const {
  isPathInside,
  resolveAppRequest,
  resolveExistingFileWithinRoot,
} = require("../dist-electron/protocol-path.js")
const { normalizePlatform } = require("../dist-electron/platform.js")

const exportRoot = path.resolve(__dirname, "..", "out")

async function main() {
  assert.equal(resolveAppRequest(exportRoot, "app://prova/"), path.join(exportRoot, "index.html"))
  assert.equal(
    resolveAppRequest(exportRoot, "app://prova/session/?from=test#ignored"),
    path.join(exportRoot, "session", "index.html"),
  )
  assert.equal(
    resolveAppRequest(exportRoot, "app://prova/_next/static/chunks/app.js"),
    path.join(exportRoot, "_next", "static", "chunks", "app.js"),
  )

  const rejected = [
    "not a URL",
    "https://prova/",
    "app://evil/",
    "app://user:pass@prova/",
    "app://prova:444/",
    "app://prova/../package.json",
    "app://prova/%2e%2e/package.json",
    "app://prova/%2E%2E%2Fpackage.json",
    "app://prova/%252e%252e%252fpackage.json",
    "app://prova/%2e%2e%5cpackage.json",
    "app://prova/%5C..%5Cpackage.json",
    "app://prova/C:%5CWindows%5Cwin.ini",
    "app://prova/C:/Windows/win.ini",
    "app://prova/index.html:stream",
    "app://prova/%00secret",
    "app://prova/%E0%A4%A",
  ]
  for (const url of rejected) {
    assert.equal(resolveAppRequest(exportRoot, url), null, `must reject ${url}`)
  }

  // Exercise Windows drive/UNC containment even when this test runs on macOS.
  assert.equal(isPathInside("C:\\app\\out", "C:\\app\\out\\index.html", path.win32), true)
  assert.equal(isPathInside("C:\\app\\out", "C:\\app\\outside.txt", path.win32), false)
  assert.equal(isPathInside("C:\\app\\out", "D:\\outside.txt", path.win32), false)
  assert.equal(isPathInside("C:\\app\\out", "\\\\server\\share\\file", path.win32), false)

  // Every generated export file, including every route and _next asset, must
  // round-trip through the protocol mapper to that exact path.
  for (const file of walkFiles(exportRoot)) {
    const relative = path.relative(exportRoot, file).split(path.sep).map(encodeURIComponent).join("/")
    assert.equal(resolveAppRequest(exportRoot, `app://prova/${relative}?v=1#asset`), file)
  }
  for (const file of walkFiles(exportRoot).filter((file) => path.basename(file) === "index.html")) {
    const relativeDirectory = path.relative(exportRoot, path.dirname(file)).split(path.sep).join("/")
    const route = relativeDirectory === "" ? "" : relativeDirectory
    assert.equal(resolveAppRequest(exportRoot, `app://prova/${route}`), file)
    assert.equal(resolveAppRequest(exportRoot, `app://prova/${route}/`), file)
  }

  const temporaryRoot = fs.mkdtempSync(path.join(os.tmpdir(), "prova-protocol-"))
  try {
    const rendererRoot = path.join(temporaryRoot, "out")
    const outsideFile = path.join(temporaryRoot, "outside.txt")
    fs.mkdirSync(rendererRoot)
    fs.writeFileSync(outsideFile, "outside")
    if (process.platform !== "win32") {
      const link = path.join(rendererRoot, "escape.txt")
      fs.symlinkSync(outsideFile, link)
      assert.equal(await resolveExistingFileWithinRoot(rendererRoot, link), null)
    }
    assert.equal(await resolveExistingFileWithinRoot(rendererRoot, path.join(rendererRoot, "missing")), null)
  } finally {
    fs.rmSync(temporaryRoot, { recursive: true, force: true })
  }

  assert.equal(normalizePlatform("darwin"), "macos")
  assert.equal(normalizePlatform("win32"), "windows")
  assert.equal(normalizePlatform("linux"), "linux")
  assert.throws(() => normalizePlatform("freebsd"), /Unsupported desktop platform/)

  console.log("Protocol path and export checks passed")
}

function walkFiles(directory) {
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const candidate = path.join(directory, entry.name)
    return entry.isDirectory() ? walkFiles(candidate) : [candidate]
  })
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
