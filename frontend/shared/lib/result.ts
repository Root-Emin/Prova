import type { RubricStatus } from "@/lib/rubric"

/** A single scored criterion: status, rationale and an evidence quote. */
export type ResultCriterion = {
  id: string
  code: string
  name: string
  weight: number
  mandatory: boolean
  status: RubricStatus
  earnedScore: number
  rationale: string
  quote: {
    speaker: string
    time: string
    text: string
  }
  /** Filled in when a trainer has overridden the score. */
  override?: {
    newStatus: RubricStatus
    reason: string
    overriddenBy: string
  }
}

export type ResultInfo = {
  sessionNo: string
  scenario: string
  persona: string
  version: string
  employee: string
  date: string
  duration: string
  evaluatorModel: string
  passThreshold: number
  device?: string
}

export function scoreForStatus(status: RubricStatus, weight: number) {
  if (status === "passed") return weight
  if (status === "partial") return Math.round(weight / 2)
  return 0
}

export function effectiveStatus(criterion: ResultCriterion): RubricStatus {
  return criterion.override?.newStatus ?? criterion.status
}

/** If a mandatory criterion fails, the result is failed regardless of the total. */
export function computeResult(criteria: ResultCriterion[], threshold: number) {
  const totalScore = criteria.reduce(
    (total, criterion) => total + criterion.earnedScore,
    0
  )
  const failedMandatory = criteria.filter(
    (criterion) => criterion.mandatory && effectiveStatus(criterion) === "failed"
  )
  const result: RubricStatus =
    failedMandatory.length > 0
      ? "failed"
      : totalScore >= threshold
        ? "passed"
        : "failed"

  return { totalScore, failedMandatory, result }
}
