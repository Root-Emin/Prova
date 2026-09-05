import { pathToFileURL } from "node:url"

import { net, protocol } from "electron"

import { resolveAppRequest, resolveExistingFileWithinRoot } from "./protocol-path"

export const APP_URL = "app://prova/"

export function registerAppScheme(): void {
  protocol.registerSchemesAsPrivileged([
    {
      scheme: "app",
      privileges: {
        standard: true,
        secure: true,
        supportFetchAPI: true,
        codeCache: true,
      },
    },
  ])
}

export async function installAppProtocol(exportRoot: string): Promise<void> {
  await protocol.handle("app", async (request) => {
    const candidate = resolveAppRequest(exportRoot, request.url)
    const filePath = candidate
      ? await resolveExistingFileWithinRoot(exportRoot, candidate)
      : null
    if (!filePath) {
      return new Response("Not found", {
        status: 404,
        headers: { "content-type": "text/plain; charset=utf-8" },
      })
    }

    return net.fetch(pathToFileURL(filePath).toString())
  })
}
