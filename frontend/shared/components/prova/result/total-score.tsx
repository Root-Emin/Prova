import { RubricStatusBadge } from "@/components/prova/rubric-status"
import type { RubricStatus } from "@/lib/rubric"
import type { ResultInfo } from "@/lib/result"

export function TotalScore({
  score,
  threshold,
  result,
  info,
}: {
  score: number
  threshold: number
  result: RubricStatus
  info: ResultInfo
}) {
  return (
    <section className="flex items-start justify-between gap-4 rounded-lg border border-line-strong bg-card px-4 py-3">
      <div>
        <h1 className="font-heading text-xl tracking-tight">{info.scenario}</h1>
        <p className="prova-meta normal-case">
          {info.employee} · {info.persona} · sürüm {info.version}
        </p>
        <p className="prova-meta normal-case">
          Oturum {info.sessionNo} · {info.date} · süre {info.duration}
        </p>
      </div>

      <div className="flex shrink-0 items-center gap-4">
        <div className="text-right whitespace-nowrap">
          <div className="font-heading text-3xl leading-none text-primary">
            {score}
            <span className="text-lg text-muted-foreground">/100</span>
          </div>
          <p className="prova-meta mt-1 whitespace-nowrap">Geçme eşiği {threshold}</p>
        </div>
        <RubricStatusBadge status={result} className="px-3 py-1 text-sm" />
      </div>
    </section>
  )
}
