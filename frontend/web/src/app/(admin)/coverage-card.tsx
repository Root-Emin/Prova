import { Progress, ProgressLabel } from "@/components/ui/progress"
import { PanelCard } from "./panel-card"

export type CoverageRow = {
  label: string
  /** Oranın okunur karşılığı: "5 / 16 çalışan". */
  note: string
  value: number
  total: number
}

/** Sertifikasyonun kurum genelindeki kapsamı; üç oran, üç ince çubuk. */
export function CoverageCard({ rows }: { rows: CoverageRow[] }) {
  return (
    <PanelCard title="Sertifikasyon kapsamı" description="Kurum geneli">
      <div className="space-y-5">
        {rows.map((row) => {
          const percent = row.total === 0 ? 0 : Math.round((row.value / row.total) * 100)
          return (
            <div key={row.label}>
              <Progress value={percent} className="gap-2">
                <ProgressLabel className="text-sm font-normal">
                  {row.label}
                </ProgressLabel>
                <span className="ml-auto text-sm font-medium tabular-nums">
                  %{percent}
                </span>
              </Progress>
              <p className="prova-meta mt-2 normal-case">{row.note}</p>
            </div>
          )
        })}
      </div>
    </PanelCard>
  )
}
