"use client"

import * as React from "react"
import Link from "next/link"
import { notFound } from "next/navigation"
import { ArrowLeft, Award } from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { ResultView } from "@/components/prova/result/result-view"
import { RubricStatusBadge } from "@/components/prova/rubric-status"
import { currentAdmin } from "@/lib/current-admin"
import type { RubricStatus } from "@/lib/rubric"
import {
  scoreForStatus,
  computeResult,
  type ResultCriterion,
} from "@/lib/result"
import { sessionDetails } from "../details"
import { TranscriptPanel } from "./transcript-panel"

export default function SessionDetailPage({
  params,
}: {
  params: Promise<{ sessionId: string }>
}) {
  const { sessionId } = React.use(params)
  const detail = sessionDetails[sessionId]
  if (!detail) notFound()

  const [criteria, setCriteria] = React.useState<ResultCriterion[]>(
    detail.criteria
  )

  function applyOverride(id: string, newStatus: RubricStatus, reason: string) {
    setCriteria((prev) =>
      prev.map((criterion) =>
        criterion.id === id
          ? {
              ...criterion,
              earnedScore: scoreForStatus(newStatus, criterion.weight),
              override: { newStatus, reason, overriddenBy: currentAdmin.name },
            }
          : criterion
      )
    )
    toast.success("Puan ezildi", {
      description: "Gerekçe denetim kaydına işlendi.",
    })
  }

  const { result } = computeResult(criteria, detail.info.passThreshold)

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <Button
            nativeButton={false}
            variant="ghost"
            size="sm"
            render={<Link href="/sessions" />}
          >
            <ArrowLeft aria-hidden />
            Oturumlar
          </Button>
          <div className="border-l border-border pl-3">
            <p className="prova-meta uppercase">Oturum incelemesi</p>
            <p className="font-mono text-xs text-muted-foreground">{detail.info.sessionNo}</p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <RubricStatusBadge status={result} />
          {result === "passed" ? (
            <Button
              onClick={() =>
                toast.success("Sertifika oluşturuldu", {
                  description: `${detail.info.employee} · ${detail.info.scenario} · sürüm ${detail.info.version}`,
                })
              }
            >
              <Award aria-hidden />
              Sertifika oluştur
            </Button>
          ) : (
            <p className="prova-meta normal-case">
              Sonuç kaldı olduğu için sertifika oluşturulamaz.
            </p>
          )}
        </div>
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_320px]">
        <main className="min-w-0 space-y-3">
          <div className="flex items-end justify-between gap-4">
            <div>
              <h1 className="font-heading text-2xl tracking-tight">Sonuç ve kriterler</h1>
              <p className="prova-meta mt-1 normal-case">
                {detail.info.employee} · {detail.info.persona} · değerlendirme ve kanıtlar
              </p>
            </div>
          </div>
          <ResultView
            info={detail.info}
            criteria={criteria}
            onOverride={applyOverride}
            canOverride
          />
        </main>
        <TranscriptPanel lines={detail.transcript} />
      </div>
    </div>
  )
}
