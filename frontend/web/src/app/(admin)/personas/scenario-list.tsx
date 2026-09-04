"use client"

import { cn } from "@/lib/utils"
import type { Scenario } from "./mock"

/** Fixed-height scenario list that scrolls inside itself. */
export function ScenarioList({
  scenarios,
  selectedId,
  onSec,
}: {
  scenarios: Scenario[]
  selectedId: string
  onSec: (id: string) => void
}) {
  const publishedCount = scenarios.filter(
    (scenario) => scenario.status === "published"
  ).length

  return (
    <nav
      aria-label="Senaryolar"
      className="prova-kart sticky top-6 flex max-h-[520px] w-[240px] shrink-0 flex-col self-start"
    >
      <div className="flex items-baseline justify-between border-b border-border px-3 py-2">
        <span className="prova-meta uppercase">Senaryolar</span>
        <span className="prova-meta normal-case">
          {publishedCount}/{scenarios.length} yayında
        </span>
      </div>

      <ul className="min-h-0 flex-1 overflow-y-auto">
        {scenarios.map((scenario) => {
          const selected = scenario.id === selectedId
          return (
            <li
              key={scenario.id}
              className="not-last:border-b not-last:border-border"
            >
              <button
                type="button"
                onClick={() => onSec(scenario.id)}
                aria-current={selected ? "true" : undefined}
                className={cn(
                  "w-full px-3 py-2 text-left transition-colors",
                  selected
                    ? "border-l-[3px] border-l-primary bg-accent pl-[9px]"
                    : "hover:bg-accent/60"
                )}
              >
                <span className="flex items-start justify-between gap-2">
                  <span
                    className={cn(
                      "text-sm leading-snug font-medium",
                      selected ? "text-primary" : "text-foreground"
                    )}
                  >
                    {scenario.name}
                  </span>
                  <span
                    className={cn(
                      "prova-meta shrink-0 rounded border px-1 normal-case",
                      scenario.status === "published"
                        ? "border-line-strong text-foreground"
                        : "border-border text-muted-foreground"
                    )}
                  >
                    {scenario.status === "published" ? "yayında" : "taslak"}
                  </span>
                </span>
                <span className="prova-meta mt-1 block normal-case">
                  {scenario.sector} · {scenario.criteria.length} kriter ·{" "}
                  {scenario.lastEditedAt}
                </span>
              </button>
            </li>
          )
        })}
      </ul>
    </nav>
  )
}
