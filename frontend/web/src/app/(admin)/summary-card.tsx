import type { LucideIcon } from "lucide-react"

import { cn } from "@/lib/utils"

/**
 * Panelin üst şeridindeki sayaç. Kart gölgesi yok; ayrım ince çizgi ve yüzey
 * farkıyla yapılır. Dikkat isteyen sayaç renkle değil kalın çerçeveyle
 * ayrışır — turuncu yüzeyin panelde işi yok.
 */
export function SummaryCard({
  title,
  value,
  description,
  icon: Icon,
  critical = false,
}: {
  title: string
  value: string
  description: string
  icon: LucideIcon
  /** Kritik çerçeve: sayaç bugün aksiyon istiyor. */
  critical?: boolean
}) {
  return (
    <div
      className={cn(
        "rounded-lg border bg-card px-4 py-3",
        critical ? "border-line-strong" : "border-border"
      )}
    >
      <div className="flex items-center gap-2">
        <span className="flex size-7 shrink-0 items-center justify-center rounded-md bg-muted">
          <Icon size={16} className="text-primary" aria-hidden />
        </span>
        <span className="prova-meta uppercase">{title}</span>
      </div>
      <p className="mt-3 font-heading text-3xl leading-none text-primary">
        {value}
      </p>
      <p className="prova-meta mt-2 normal-case">{description}</p>
    </div>
  )
}
