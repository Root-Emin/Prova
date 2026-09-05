import { MicOff } from "lucide-react"

import { cn } from "@/lib/utils"

/**
 * The product's core selling point. A visible, set-apart block, not a footnote.
 * Appears on the register screen, on the last desktop intro screen, and on the
 * session briefing screen.
 */
function PrivacyNotice({ className }: { className?: string }) {
  return (
    <section
      className={cn(
        "flex gap-3 rounded-lg border border-line-strong bg-tint px-4 py-3",
        className
      )}
    >
      <MicOff size={20} className="mt-0.5 shrink-0 text-primary" aria-hidden />
      <div>
        <p className="font-heading text-sm font-semibold text-primary">
          Sesiniz cihazdan çıkmaz
        </p>
        <p className="mt-1 text-sm leading-relaxed text-foreground">
          Konuşma ve transkript cihazınızda işlenir. Ham ses kaydı saklanmaz,
          sunucuya gönderilmez. Kuruma yalnızca puan ve gerekçe ulaşır.
        </p>
      </div>
    </section>
  )
}

export { PrivacyNotice }
