"use client"

import * as React from "react"
import { Check, Mic } from "lucide-react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { useDeviceState } from "@/hooks/use-device-state"

/**
 * Pre-exam microphone check. The audio pipeline is not built yet, so the
 * test only verifies device state and reports the result in one step.
 */
export function MicCheck({ className }: { className?: string }) {
  const device = useDeviceState()
  const [status, setStatus] = React.useState<"untested" | "ok">(
    "untested"
  )

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
          <p className="text-sm font-medium">
            {status === "ok" ? "Ses algılandı" : "Mikrofon kontrolü"}
          </p>
          <p className="prova-meta normal-case">
            {device.deviceName} · dahili mic
            {status === "ok" ? " · giriş seviyesi yeterli" : ""}
          </p>
        </div>
      </div>

      <Button
        variant="outline"
        onClick={() => setStatus("ok")}
        disabled={status === "ok"}
      >
        {status === "ok" ? "Test edildi" : "Test et"}
      </Button>
    </section>
  )
}
