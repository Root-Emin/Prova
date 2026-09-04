"use client"

import type { ModelUsage } from "./mock"

/**
 * A single bar chart. No charting library is pulled in; the bars are drawn
 * with surface widths. The slate is the brand colour, it carries no status.
 */
export function RoutingChart({ data }: { data: ModelUsage[] }) {
  const max = Math.max(...data.map((line) => line.session))

  return (
    <section className="rounded-lg border border-border bg-card p-5">
      <h2 className="text-sm font-medium">Model başına oturum</h2>
      <p className="prova-meta normal-case">Son 30 gün</p>

      <ul className="mt-5 space-y-4">
        {data.map((line) => (
          <li key={`${line.model}-${line.task}`}>
            <div className="flex items-baseline justify-between gap-4">
              <span className="font-mono text-xs">{line.model}</span>
              <span className="prova-meta normal-case">
                {line.task} · {line.session} oturum
              </span>
            </div>
            <div className="mt-1.5 h-3 w-full rounded-sm bg-muted">
              <div
                className={
                  line.task === "Konuşma"
                    ? "h-3 rounded-sm bg-primary"
                    : "h-3 rounded-sm bg-brand-hover"
                }
                style={{ width: `${(line.session / max) * 100}%` }}
              />
            </div>
          </li>
        ))}
      </ul>
    </section>
  )
}
