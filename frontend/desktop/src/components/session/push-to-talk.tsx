import { Mic, MicOff } from "lucide-react"

import { cn } from "@/lib/utils"

/**
 * Push-to-talk indicator. Not a button: on the exam screen space never
 * triggers a control, it only opens the microphone.
 */
export function PushToTalk({
  speaking,
  characterReplying,
  finished,
}: {
  speaking: boolean
  characterReplying: boolean
  finished: boolean
}) {
  const status = finished
    ? "finished"
    : speaking
      ? "recording"
      : characterReplying
        ? "character"
        : "ready"

  const text = {
    ready: "Konuşmak için space tuşunu basılı tutun",
    recording: "Kayıtta — bırakınca gönderilir",
    character: "Karakter yanıtlıyor",
    finished: "Görüşme tamamlandı",
  }[status]

  return (
    <div
      data-status={status}
      aria-live="polite"
      className={cn(
        "flex items-center justify-center gap-3 rounded-lg border px-4 py-3",
        status === "recording"
          ? "border-primary bg-tint"
          : "border-border bg-card"
      )}
    >
      {status === "recording" ? (
        <Mic size={20} className="text-primary" aria-hidden />
      ) : (
        <MicOff size={20} className="text-muted-foreground" aria-hidden />
      )}
      <span
        className={cn(
          "text-sm font-medium",
          status === "recording" ? "text-primary" : "text-muted-foreground"
        )}
      >
        {text}
      </span>
      <kbd className="prova-meta rounded border border-line-strong bg-background px-1.5 py-0.5 font-mono">
        space
      </kbd>
    </div>
  )
}
