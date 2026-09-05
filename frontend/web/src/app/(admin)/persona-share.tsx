import { cn } from "@/lib/utils"
import { PanelCard } from "./panel-card"
import {
  personaColorClass,
  personaColors,
  type PersonaColor,
} from "./personas/persona-colors"

export type PersonaShare = {
  color: PersonaColor
  sessions: number
}

/**
 * Oturumların persona kişiliklerine dağılımı. Persona rengi paletin tek
 * istisnasıdır ve yalnızca kimlik rozetinde görünür; çubuk arduvazdır,
 * renk burada durum anlatmaz.
 */
export function PersonaShareCard({ rows }: { rows: PersonaShare[] }) {
  const total = rows.reduce((sum, row) => sum + row.sessions, 0)

  return (
    <PanelCard title="Persona oturum dağılımı" description={`${total} oturum`}>
      <ul className="space-y-4">
        {rows.map((row) => {
          const persona = personaColors[row.color]
          const renk = personaColorClass[row.color]
          const oran = total === 0 ? 0 : (row.sessions / total) * 100

          return (
            <li key={row.color} className="flex items-center gap-3">
              <span
                className={cn(
                  "flex size-6 shrink-0 items-center justify-center rounded-full border text-xs font-medium",
                  renk.soft,
                  renk.ink,
                  renk.border
                )}
                aria-hidden
              >
                {persona.title.slice(0, 1)}
              </span>
              <span className="w-[68px] shrink-0 truncate text-sm">
                {persona.title}
              </span>
              <span className="h-1.5 min-w-0 flex-1 rounded-full bg-muted">
                <span
                  className="block h-1.5 rounded-full bg-primary"
                  style={{ width: `${oran}%` }}
                />
              </span>
              <span className="w-8 shrink-0 text-right font-medium tabular-nums">
                {row.sessions}
              </span>
            </li>
          )
        })}
      </ul>
    </PanelCard>
  )
}
