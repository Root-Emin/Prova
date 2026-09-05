import { contextBridge, ipcRenderer } from "electron"

import type { ProvaAPI } from "./api-types"

// A sandboxed preload cannot require arbitrary local modules at runtime.
// Keep these compile-time constants in this file so the emitted preload is
// self-contained; the main process owns the matching allow-list.
const IPC_CHANNELS = {
  systemGetInfo: "prova:system:get-info",
  deviceGetInfo: "prova:device:get-info",
  deviceGetRegistrationInfo: "prova:device:get-registration-info",
  permissionsGetMicrophoneStatus: "prova:permissions:get-microphone-status",
  authRequestLoginCode: "prova:auth:request-login-code",
  authVerifyLoginCode: "prova:auth:verify-login-code",
  onboardingGetState: "prova:onboarding:get-state",
  onboardingComplete: "prova:onboarding:complete",
} as const

const api: ProvaAPI = Object.freeze({
  system: Object.freeze({
    getInfo: () => ipcRenderer.invoke(IPC_CHANNELS.systemGetInfo),
  }),
  device: Object.freeze({
    getInfo: () => ipcRenderer.invoke(IPC_CHANNELS.deviceGetInfo),
    getRegistrationInfo: () => ipcRenderer.invoke(IPC_CHANNELS.deviceGetRegistrationInfo),
  }),
  permissions: Object.freeze({
    getMicrophoneStatus: () => ipcRenderer.invoke(IPC_CHANNELS.permissionsGetMicrophoneStatus),
  }),
  auth: Object.freeze({
    requestLoginCode: (email: string) => ipcRenderer.invoke(IPC_CHANNELS.authRequestLoginCode, email),
    verifyLoginCode: (email: string, code: string) => ipcRenderer.invoke(IPC_CHANNELS.authVerifyLoginCode, email, code),
  }),
  onboarding: Object.freeze({
    getState: () => ipcRenderer.invoke(IPC_CHANNELS.onboardingGetState),
    complete: () => ipcRenderer.invoke(IPC_CHANNELS.onboardingComplete),
  }),
})

contextBridge.exposeInMainWorld("prova", api)
