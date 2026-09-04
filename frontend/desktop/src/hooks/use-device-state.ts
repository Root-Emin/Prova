"use client"

import * as React from "react"

/**
 * Device and microphone state. This will be wired to Electron IPC later;
 * for now it returns a mock. Screens never reach device data except here.
 */
export type MicStatus = "ready" | "muted" | "no-permission"

export type DeviceState = {
  orgName: string
  deviceName: string | null
  os: string
  registeredAt: string | null
  registered: boolean
  mic: MicStatus
}

const mockDevice: DeviceState = {
  orgName: "Anadolu Katılım Bankası",
  deviceName: "SUBE-42-IST",
  os: "Windows 11 23H2",
  registeredAt: "11.04.2026",
  registered: true,
  mic: "ready",
}

export function useDeviceState(): DeviceState {
  const [status] = React.useState<DeviceState>(mockDevice)
  return status
}
