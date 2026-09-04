"use client"

import { CriteriaAccordion } from "@/components/prova/result/criteria-accordion"
import { TotalScore } from "@/components/prova/result/total-score"
import { MandatoryBanner } from "@/components/prova/result/mandatory-banner"
import type { RubricStatus } from "@/lib/rubric"
import { computeResult, type ResultInfo, type ResultCriterion } from "@/lib/result"

/**
 * The single body of a scoring result. The employee view on desktop and the
 * trainer review screen on web both render this component.
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

      <CriteriaAccordion
        criteria={criteria}
        onOverride={onOverride}
        canOverride={canOverride}
      />

      <p className="prova-meta normal-case">
        Değerlendiren model {info.evaluatorModel}
        {info.device ? ` · cihaz ${info.device}` : ""}. Her puanın yanında
        transkriptten kanıt alıntısı bulunur.
      </p>
    </div>
  )
}
