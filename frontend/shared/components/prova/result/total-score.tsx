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
    <section className="overflow-hidden rounded-xl border border-line-strong bg-card">
      <div className="flex flex-col gap-5 px-5 py-5 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <p className="prova-meta uppercase">Değerlendirme özeti</p>
            <span className="prova-meta font-mono normal-case">{info.sessionNo}</span>
          </div>
          <h1 className="mt-2 font-heading text-2xl tracking-tight">{info.scenario}</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {info.employee} · {info.persona}
          </p>
        </div>

        <div className="flex shrink-0 items-center gap-4 sm:pt-1">
          <div className="text-right whitespace-nowrap">
            <div className="font-heading text-4xl leading-none text-primary">
              {score}
              <span className="text-lg text-muted-foreground">/100</span>
            </div>
            <p className="prova-meta mt-1 whitespace-nowrap">
              Geçme eşiği {threshold}
            </p>
          </div>
          <RubricStatusBadge status={result} className="px-3 py-1.5 text-sm" />
        </div>
      </div>

      <dl className="grid grid-cols-2 border-t border-border bg-muted/50 sm:grid-cols-4">
        <div className="border-b border-border px-5 py-3 sm:border-r sm:border-b-0">
          <dt className="prova-meta uppercase">Sürüm</dt>
          <dd className="mt-1 text-sm font-medium">{info.version}</dd>
        </div>
        <div className="border-b border-border px-5 py-3 sm:border-r sm:border-b-0">
          <dt className="prova-meta uppercase">Tarih</dt>
          <dd className="mt-1 text-sm font-medium">{info.date}</dd>
        </div>
        <div className="border-r border-border px-5 py-3">
          <dt className="prova-meta uppercase">Süre</dt>
          <dd className="mt-1 font-mono text-sm font-medium">{info.duration}</dd>
        </div>
        <div className="px-5 py-3">
          <dt className="prova-meta uppercase">Cihaz</dt>
          <dd className="mt-1 truncate font-mono text-sm font-medium">
            {info.device ?? "—"}
          </dd>
        </div>
      </dl>
    </section>
  )
}
