import type { LucideIcon } from "lucide-react"

import { PanelCard } from "./panel-card"

export type VerificationRow = {
  label: string
  value: string
  icon: LucideIcon
}

/** Kimlik zincirinin panel özeti: hesap doğrulaması, kod, cihaz, ret. */
export function VerificationCard({ rows }: { rows: VerificationRow[] }) {
  return (
    <PanelCard title="Doğrulama ve cihaz" description="Kimlik zinciri">
      <ul className="space-y-4">
        {rows.map((row) => (
          <li key={row.label} className="flex items-center gap-3">
            <row.icon size={16} className="shrink-0 text-primary" aria-hidden />
            <span className="min-w-0 flex-1 text-sm">{row.label}</span>
            <span className="font-medium tabular-nums">{row.value}</span>
          </li>
        ))}
      </ul>
    </PanelCard>
  )
}
