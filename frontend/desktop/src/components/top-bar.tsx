"use client"

import { Laptop, Mic, MicOff, TriangleAlert } from "lucide-react"

import { cn } from "@/lib/utils"
import { useDeviceState } from "@/hooks/use-device-state"

const micView = {
  ready: { icon: Mic, label: "Mikrofon hazır" },
  muted: { icon: MicOff, label: "Mikrofon kapalı" },
  "no-permission": { icon: TriangleAlert, label: "Mikrofon izni yok" },
} as const

/**
 * Electron shell: no sidebar, just a thin top bar. The bar doubles as the
 * drag region of the frameless window, and simplifies once the exam starts.
 */
export function TopBar({ sade = false }: { sade?: boolean }) {
  const device = useDeviceState()
  const mic = micView[device.mic]
  const MicIcon = mic.icon

  return (
    <header
      data-slot="ust-bar"
      className={cn(
        "prova-drag flex h-11 shrink-0 items-center justify-between border-b border-border bg-card px-4",
        sade && "bg-background"
      )}
    >
      <div className="flex items-center gap-3">
        <span className="font-heading text-sm tracking-tight text-primary">
          Prova
        </span>
        {!sade && (
          <>
            <span className="h-4 w-px bg-border" aria-hidden />
            <span className="text-sm text-foreground">{device.orgName}</span>
          </>
        )}
      </div>

      <div className="flex items-center gap-4">
        {!sade && device.registered && (
          <span className="inline-flex items-center gap-1.5 rounded-md border border-border bg-background px-2 py-1">
            <Laptop size={16} className="text-primary" aria-hidden />
            <span className="prova-meta">Kayıtlı cihaz · {device.deviceName}</span>
          </span>
        )}
        <span className="inline-flex items-center gap-1.5">
          <MicIcon size={16} className="text-muted-foreground" aria-hidden />
          <span className="prova-meta">{mic.label}</span>
        </span>
      </div>
    </header>
  )
}
