import { WifiOff } from "lucide-react"

/**
 * Blocking overlay drawn over the transcript when the connection drops.
 * The conversation pauses; the exam timer keeps running.
 */
export function ConnectionOverlay() {
  return (
    <div
      role="alert"
      className="absolute inset-0 z-10 flex items-center justify-center bg-background/85 backdrop-blur-[1px]"
    >
      <div className="w-[380px] rounded-lg border border-line-strong bg-card px-4 py-3 text-center">
        <WifiOff size={20} className="mx-auto text-foreground" aria-hidden />
        <p className="mt-2 font-heading text-base">Bağlantı koptu</p>
        <p className="mt-1 text-sm leading-relaxed text-muted-foreground">
          Konuşma duraklatıldı, süre işlemiyor. Bağlantı geri geldiğinde
          kaldığınız yerden devam edeceksiniz.
        </p>
      </div>
    </div>
  )
}
