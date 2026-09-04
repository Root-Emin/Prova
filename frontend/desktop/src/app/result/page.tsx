"use client"

import * as React from "react"
import Link from "next/link"
import { ArrowLeft, Printer } from "lucide-react"

import { Button } from "@/components/ui/button"
import { ResultView } from "@/components/prova/result/result-view"
import { TopBar } from "@/components/top-bar"
import type { RubricStatus } from "@/lib/rubric"
import { scoreForStatus, type ResultCriterion } from "@/lib/result"
import { resultInfo, resultCriteria as initialCriteria } from "./mock"

export default function ResultPage() {
  const [criteria, setCriteria] =
    React.useState<ResultCriterion[]>(initialCriteria)

  function applyOverride(id: string, newStatus: RubricStatus, reason: string) {
    setCriteria((prev) =>
      prev.map((criterion) =>
        criterion.id === id
          ? {
              ...criterion,
              earnedScore: scoreForStatus(newStatus, criterion.weight),
              override: { newStatus, reason, overriddenBy: "Burak Yıldırım · eğitmen" },
            }
          : criterion
      )
    )
  }

  return (
    <div className="prova-yazdirilabilir flex h-full flex-col overflow-hidden">
      <TopBar />

      <div className="flex min-h-0 flex-1 flex-col gap-4 p-5">
        <div className="min-h-0 flex-1 overflow-y-auto">
          <ResultView
            info={resultInfo}
            criteria={criteria}
            onOverride={applyOverride}
          />
        </div>

        <div className="prova-yazdirma-disi flex items-center justify-between border-t border-border pt-4">
          <Button nativeButton={false} variant="ghost" render={<Link href="/" />}>
            <ArrowLeft aria-hidden />
            Oturumlara dön
          </Button>
          <Button variant="outline" onClick={() => window.print()}>
            <Printer aria-hidden />
            Sertifikayı yazdır
          </Button>
        </div>
      </div>
    </div>
  )
}
