import type { ProvaAPI } from "../../electron/api-types"

declare global {
  interface Window {
    readonly prova?: ProvaAPI
  }
}

export {}
