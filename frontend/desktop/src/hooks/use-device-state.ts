"use client"

import * as React from "react"

import type { NormalizedPlatform } from "../../electron/api-types"

/**
 * Renderer-facing device state. Electron details arrive only through the
 * narrow preload bridge; React code never imports Electron itself.
 */
export type MicStatus = "ready" | "muted" | "no-permission"

export type DeviceState = {
  orgName: string
  deviceName: string | null
  os: string
  platform: NormalizedPlatform | null
  appVersion: string | null
  fingerprint: string | null
  registeredAt: string | null
  registered: boolean
  mic: MicStatus
}

const initialDevice: DeviceState = {
  orgName: "Anadolu Katılım Bankası",
  deviceName: null,
  os: "İşletim sistemi bilgisi alınıyor",
  platform: null,
  appVersion: null,
  fingerprint: null,
  registeredAt: null,
  // Registration is backend-owned and must not be inferred from a local key.
  registered: false,
  mic: "muted",
}

export function useDeviceState(): DeviceState {
  const [status, setStatus] = React.useState<DeviceState>(initialDevice)

  React.useEffect(() => {
    const api = window.prova
    if (!api) return

    let active = true

    const loadDevice = async () => {
      const [info, registration, permission] = await Promise.allSettled([
        api.device.getInfo(),
        api.device.getRegistrationInfo(),
        api.permissions.getMicrophoneStatus(),
      ])
      if (!active) return

      setStatus((current) => {
        const deviceInfo = info.status === "fulfilled" ? info.value : null
        return {
          ...current,
          deviceName: deviceInfo?.displayName ?? current.deviceName,
          os: deviceInfo ? formatOs(deviceInfo.platform, deviceInfo.osRelease) : current.os,
          platform: deviceInfo?.platform ?? current.platform,
          appVersion: deviceInfo?.appVersion ?? current.appVersion,
          fingerprint: registration.status === "fulfilled"
            ? registration.value.fingerprint
            : current.fingerprint,
          mic: permission.status === "fulfilled"
            ? toMicStatus(permission.value)
            : current.mic,
        }
      })
    }

    void loadDevice()
    window.addEventListener("focus", loadDevice)
    window.addEventListener("prova:microphone-status-changed", loadDevice)
    return () => {
      active = false
      window.removeEventListener("focus", loadDevice)
      window.removeEventListener("prova:microphone-status-changed", loadDevice)
    }
  }, [])

  return status
}

function formatOs(platform: NormalizedPlatform, release: string): string {
  const label = platform === "macos" ? "macOS" : platform === "windows" ? "Windows" : "Linux"
  return `${label} ${release}`
}

function toMicStatus(status: Awaited<ReturnType<NonNullable<Window["prova"]>["permissions"]["getMicrophoneStatus"]>>): MicStatus {
  if (status === "granted") return "ready"
  if (status === "denied" || status === "restricted") return "no-permission"
  return "muted"
}
