"use client"

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion"
import { RubricStatusInline } from "@/components/prova/rubric-status"
import { TranscriptQuote } from "@/components/prova/transcript-quote"
import { rubricStatusView, type RubricStatus } from "@/lib/rubric"
import { ScoreOverrideDialog } from "@/components/prova/result/score-override-dialog"
import { effectiveStatus, type ResultCriterion } from "@/lib/result"

export function CriteriaAccordion({
  criteria,
  onOverride,
  canOverride = true,
}: {
  criteria: ResultCriterion[]
  onOverride?: (id: string, newStatus: RubricStatus, reason: string) => void
  /** Only trainers and admins may override a score. */
  canOverride?: boolean
}) {
  return (
    <Accordion
      multiple={false}
      className="rounded-lg border border-border bg-card px-4"
    >
      {criteria.map((criterion) => {
        const status = effectiveStatus(criterion)
        return (
          <AccordionItem key={criterion.id} value={criterion.id}>
            <AccordionTrigger className="gap-4">
              <span className="flex flex-1 items-center gap-3">
                <span className="prova-meta w-[68px] shrink-0 uppercase">
                  {criterion.code}
                </span>
                <span className="flex-1 text-left">{criterion.name}</span>
                <RubricStatusInline status={status} />
                <span className="prova-meta w-[76px] shrink-0 text-right normal-case">
                  {criterion.earnedScore}/{criterion.weight}
                </span>
              </span>
            </AccordionTrigger>

            <AccordionContent keepMounted className="space-y-4 pr-8 pl-[80px]">
              <div>
                <p className="prova-meta uppercase">Gerekçe</p>
                <p className="mt-1 text-sm leading-relaxed text-foreground">
                  {criterion.rationale}
                </p>
              </div>

              <div>
                <p className="prova-meta uppercase">Transkriptten kanıt</p>
                <TranscriptQuote
                  className="mt-1"
                  speaker={criterion.quote.speaker}
                  timestamp={criterion.quote.time}
                >
                  {criterion.quote.text}
                </TranscriptQuote>
              </div>

              {criterion.override ? (
                <div className="rounded-md border border-line-strong bg-muted px-3 py-2">
                  <p className="prova-meta uppercase">
                    Eğitmen ezdi · {criterion.override.overriddenBy}
                  </p>
                  <p className="mt-1 text-sm text-foreground">
                    Yeni status: {rubricStatusView[criterion.override.newStatus].label}.{" "}
                    {criterion.override.reason}
                  </p>
                </div>
              ) : null}

              <div className="flex items-center justify-between">
                <p className="prova-meta normal-case">
                  {criterion.mandatory
                    ? "Zorunlu kriter — düşerse sonuç kaldıdır."
                    : "Ağırlıklı kriter."}
                </p>
                {canOverride && onOverride ? (
                  <ScoreOverrideDialog
                    criterionCode={criterion.code}
                    criterionName={criterion.name}
                    currentStatus={status}
                    onOverride={(newStatus, reason) =>
                      onOverride(criterion.id, newStatus, reason)
                    }
                  />
                ) : null}
              </div>
            </AccordionContent>
          </AccordionItem>
        )
      })}
    </Accordion>
  )
}
