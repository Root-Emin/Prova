"use client"

import * as React from "react"
import Link from "next/link"
import { notFound } from "next/navigation"
import { ArrowLeft, Award } from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { ResultView } from "@/components/prova/result/result-view"
import { useRole } from "@/components/admin/role-context"
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

  const { role } = useRole()
  const [criteria, setCriteria] = React.useState<ResultCriterion[]>(
    detail.criteria
  )

  // Only trainers and admins may override a score; employees can only read it.
  const canOverride = role === "trainer" || role === "org-admin"

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
    toast.success("Puan ezildi", {
      description: "Gerekçe denetim kaydına işlendi.",
    })
  }

  const { result } = computeResult(criteria, detail.info.passThreshold)

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <Button
          nativeButton={false}
          variant="ghost"
          size="sm"
          render={<Link href="/sessions" />}
        >
          <ArrowLeft aria-hidden />
          Oturumlar
        </Button>

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

      <div className="flex gap-6">
        <div className="min-w-0 flex-1">
          <ResultView
            info={detail.info}
            criteria={criteria}
            onOverride={applyOverride}
            canOverride={canOverride}
          />
        </div>
        <TranscriptPanel lines={detail.transcript} />
      </div>
    </div>
  )
}
