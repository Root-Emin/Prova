import { RubricStatusInline } from "@/components/prova/rubric-status"
import type { RubricStatus } from "@/lib/rubric"
import type { SessionCriterion } from "@/app/session/mock"

/**
 * Live rubric. Every criterion starts neutral grey when the session opens.
 * Status is always conveyed by colour + icon + label together.
 */
export function RubricPanel({
  criteria,
  rubric,
}: {
  criteria: SessionCriterion[]
  rubric: Record<string, RubricStatus>
}) {
  return (
    <aside className="flex w-[320px] shrink-0 flex-col rounded-lg border border-border bg-card">
      <div className="border-b border-border px-4 py-3">
        <h2 className="text-sm font-medium">Rubrik</h2>
        <p className="prova-meta normal-case">
          Puan görüşme bitince güçlü model tarafından verilir.
        </p>
      </div>

      <ul className="min-h-0 flex-1 divide-y divide-border overflow-y-auto">
        {criteria.map((criterion) => {
          const status = rubric[criterion.id] ?? "unevaluated"
          return (
            <li key={criterion.id} className="space-y-1.5 px-4 py-3">
              <div className="prova-meta flex items-center justify-between uppercase">
                <span>{criterion.code}</span>
                <span>
                  {criterion.mandatory ? "Zorunlu · " : ""}
                  {criterion.weight} puan
                </span>
              </div>
              <p className="text-sm leading-snug text-foreground">{criterion.name}</p>
              <RubricStatusInline status={status} />
            </li>
          )
        })}
      </ul>
    </aside>
  )
}
