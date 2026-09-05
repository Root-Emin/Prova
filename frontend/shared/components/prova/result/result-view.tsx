"use client"

import { CriteriaAccordion } from "@/components/prova/result/criteria-accordion"
import { TotalScore } from "@/components/prova/result/total-score"
import { MandatoryBanner } from "@/components/prova/result/mandatory-banner"
import type { RubricStatus } from "@/lib/rubric"
import { computeResult, type ResultInfo, type ResultCriterion } from "@/lib/result"

/**
 * Shared result body for the admin review and certificate screens.
 */
export function ResultView({
  info,
  criteria,
  onOverride,
  canOverride = true,
}: {
  info: ResultInfo
  criteria: ResultCriterion[]
  onOverride?: (id: string, newStatus: RubricStatus, reason: string) => void
  canOverride?: boolean
}) {
  const { totalScore, failedMandatory, result } = computeResult(
    criteria,
    info.passThreshold
  )

  return (
    <div className="space-y-4">
      {failedMandatory.length > 0 && (
        <MandatoryBanner
          failedCriteria={failedMandatory.map((criterion) => ({
            code: criterion.code,
            name: criterion.name,
          }))}
        />
      )}

      <TotalScore
        score={totalScore}
        threshold={info.passThreshold}
        result={result}
        info={info}
      />

      <section className="space-y-3">
        <div className="flex items-end justify-between gap-4">
          <div>
            <h2 className="font-heading text-xl tracking-tight">
              Kriter değerlendirmesi
            </h2>
            <p className="prova-meta mt-1 normal-case">
              Her kriter transkript kanıtı ve gerekçesiyle birlikte incelenebilir.
            </p>
          </div>
          <span className="prova-meta shrink-0 normal-case">
            {criteria.length} kriter
          </span>
        </div>
        <CriteriaAccordion
          criteria={criteria}
          onOverride={onOverride}
          canOverride={canOverride}
        />
      </section>

      <p className="prova-meta normal-case">
        Değerlendiren model {info.evaluatorModel}
        {info.device ? ` · cihaz ${info.device}` : ""}. Her puanın yanında
        transkriptten kanıt alıntısı bulunur.
      </p>
    </div>
  )
}
