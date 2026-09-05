import Link from "next/link"
import { ChevronRight, type LucideIcon } from "lucide-react"

import { cn } from "@/lib/utils"
import { PanelCard } from "./panel-card"

export type PriorityAction = {
  title: string
  /** Satırın altındaki tek cümlelik gerekçe. */
  detail: string
  count: number
  href: string
  icon: LucideIcon
  /** Bugün aksiyon isteyen satır; kalın çerçeveyle ayrışır. */
  critical?: boolean
}

/**
 * Panelin "bir sonraki iş" listesi. Sıralama sabittir: sertifika, oturum,
 * atama, hesap, denetim. Sayısı sıfır olan satır listeye girmez.
 */
export function PriorityActions({ items }: { items: PriorityAction[] }) {
  const rows = items.filter((item) => item.count > 0)

  return (
    <PanelCard
      title="Öncelikli aksiyonlar"
      description="Kapatılmayı bekleyen kayıtlar"
      className="lg:col-span-2"
      bodyClassName="p-0"
    >
      {rows.length === 0 ? (
        <p className="prova-meta p-4 normal-case">
          Bekleyen aksiyon yok. Sertifika, atama ve hesap kayıtları güncel.
        </p>
      ) : (
        <ul>
          {rows.map((row) => (
            <li key={row.href} className="border-b border-border last:border-b-0">
              <Link
                href={row.href}
                className="flex items-center gap-3 px-4 py-3 transition-colors hover:bg-muted"
              >
                <span
                  className={cn(
                    "flex size-7 shrink-0 items-center justify-center rounded-md border bg-muted",
                    row.critical ? "border-line-strong" : "border-transparent"
                  )}
                >
                  <row.icon size={16} className="text-primary" aria-hidden />
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block text-sm font-medium">{row.title}</span>
                  <span className="prova-meta block normal-case">
                    {row.detail}
                  </span>
                </span>
                <span className="font-heading text-lg leading-none tabular-nums">
                  {row.count}
                </span>
                <ChevronRight
                  size={16}
                  className="shrink-0 text-muted-foreground"
                  aria-hidden
                />
              </Link>
            </li>
          ))}
        </ul>
      )}
    </PanelCard>
  )
}
