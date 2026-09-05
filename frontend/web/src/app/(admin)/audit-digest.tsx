import { PanelCard } from "./panel-card"
import type { AuditEntry } from "./audit-log/mock"

/** Denetim kaydının son satırları; tam kayıt kendi ekranında durur. */
export function AuditDigest({ rows }: { rows: AuditEntry[] }) {
  return (
    <PanelCard
      title="Denetim özeti"
      description="İzlenebilirlik ve son yönetim kayıtları"
      href="/audit-log"
      bodyClassName="p-0"
    >
      <ul>
        {rows.map((row) => (
          <li
            key={row.id}
            className="flex items-baseline justify-between gap-4 border-b border-border px-4 py-3 last:border-b-0"
          >
            <div className="min-w-0">
              <p className="truncate text-sm">{row.action}</p>
              <p className="prova-meta mt-1 truncate normal-case">
                {row.actor} · <span className="font-mono">{row.target}</span>
              </p>
            </div>
            <span className="prova-meta shrink-0 normal-case tabular-nums">
              {row.time}
            </span>
          </li>
        ))}
      </ul>
    </PanelCard>
  )
}
