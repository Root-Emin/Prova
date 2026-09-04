import { X } from "lucide-react"

/**
 * Banner shown at the top when a mandatory criterion fails. The result is
 * then "failed" regardless of the total score.
 */
export function MandatoryBanner({
  failedCriteria,
}: {
  failedCriteria: { code: string; name: string }[]
}) {
  return (
    <div
      role="alert"
      className="flex items-start gap-3 rounded-lg border border-fail bg-fail px-4 py-3"
    >
      <X size={20} className="mt-0.5 shrink-0 text-foreground" aria-hidden />
      <div>
        <p className="font-heading text-sm font-semibold text-foreground">
          Zorunlu kriter düştü — sonuç: kaldı
        </p>
        <p className="mt-1 text-sm leading-relaxed text-foreground">
          Aşağıdaki zorunlu kriter karşılanmadığı için oturum, toplam puandan
          bağımsız olarak kaldı sayılır.
        </p>
        <ul className="mt-2 space-y-1">
          {failedCriteria.map((criterion) => (
            <li key={criterion.code} className="text-sm text-foreground">
              <span className="prova-meta text-foreground uppercase">{criterion.code}</span>{" "}
              {criterion.name}
            </li>
          ))}
        </ul>
      </div>
    </div>
  )
}
