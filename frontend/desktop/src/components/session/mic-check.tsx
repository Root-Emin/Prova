"use client"

import * as React from "react"
import { Check, Mic } from "lucide-react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { useDeviceState } from "@/hooks/use-device-state"

/**
 * Pre-exam microphone check. This requests audio only and immediately stops
 * the stream; capture/transcription is deliberately outside this foundation.
 */
export function MicCheck({ className }: { className?: string }) {
  const device = useDeviceState()
  const [status, setStatus] = React.useState<"untested" | "testing" | "ok" | "error">(
    "untested"
  )

  const testMicrophone = async () => {
    setStatus("testing")
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: false })
      const hasLiveAudio = stream.getAudioTracks().some((track) => track.readyState === "live")
      stream.getTracks().forEach((track) => track.stop())
      setStatus(hasLiveAudio ? "ok" : "error")
    } catch {
      setStatus("error")
    } finally {
      window.dispatchEvent(new Event("prova:microphone-status-changed"))
    }
  }

  return (
    <section
      className={cn(
        "flex items-center justify-between gap-4 rounded-lg border border-border bg-card px-4 py-3",
        className
      )}
    >
      <div className="flex items-center gap-3">
        {status === "ok" ? (
          <Check size={20} className="text-primary" aria-hidden />
        ) : (
          <Mic size={20} className="text-muted-foreground" aria-hidden />
        )}
        <div>
          <p className="text-sm font-medium">{micStatusLabel(status)}</p>
          <p className="prova-meta normal-case">
            {device.deviceName} · dahili mic
            {status === "ok" ? " · giriş seviyesi yeterli" : ""}
          </p>
        </div>
      </div>

      <Button
        variant="outline"
        onClick={testMicrophone}
        disabled={status === "ok" || status === "testing"}
      >
        {status === "ok" ? "Test edildi" : status === "testing" ? "Test ediliyor" : "Test et"}
      </Button>
    </section>
  )
}

function micStatusLabel(status: "untested" | "testing" | "ok" | "error"): string {
  if (status === "ok") return "Mikrofon erişimi hazır"
  if (status === "error") return "Mikrofona erişilemedi"
  if (status === "testing") return "Mikrofon kontrol ediliyor"
  return "Mikrofon kontrolü"
}
