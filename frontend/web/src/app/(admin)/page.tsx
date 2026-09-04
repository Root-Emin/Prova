"use client"

import Link from "next/link"
import { Award, ChevronRight, ClipboardList, MessagesSquare, Percent } from "lucide-react"

import { Button } from "@/components/ui/button"
import { PageHeader } from "@/components/prova/page-header"
import { assignments } from "./assignments/mock"
import { sessionSummaries } from "./sessions/mock"
import { SessionTable } from "./sessions/session-table"
import { needsRenewal } from "./certificates/mock"
import { dashboardSummary } from "./mock"
import { SummaryCard } from "./summary-card"

export default function DashboardPage() {
  const toRenew = needsRenewal()
  const pendingList = assignments.filter(
    (assignment) => assignment.status !== "completed"
  )

  return (
    <div className="space-y-6">
      <PageHeader
        title="Panel"
        description="Kurumun sertifikasyon durumu. Son güncelleme 03.09.2026 09:44."
      />

      <div className="grid grid-cols-4 gap-4">
        <SummaryCard
          title="Bu hafta oturum"
          value={String(dashboardSummary.sessionsThisWeek)}
          description="İkisi hâlâ değerlendiriliyor"
          icon={MessagesSquare}
        />
        <SummaryCard
          title="Geçme oranı"
          value={dashboardSummary.passRate}
          description="Zorunlu kriter düşenler dahil"
          icon={Percent}
        />
        <SummaryCard
          title="Bekleyen atama"
          value={String(pendingList.length)}
          description="Biri son tarihi geçti"
          icon={ClipboardList}
        />
        <SummaryCard
          title="Yenilenecek sertifika"
          value={String(toRenew.length)}
          description="Rubrik sürümü eskidi"
          icon={Award}
        />
      </div>

      {toRenew.length > 0 && (
        <section className="flex items-center justify-between gap-6 rounded-lg border border-line-strong bg-sage px-4 py-3">
          <div>
            <p className="font-heading text-sm font-semibold text-primary">
              Mevzuat değişti, {toRenew.length} sertifika eski rubrikle
              verildi
            </p>
            <p className="mt-1 text-sm text-foreground">
              Etkilenen personeli toplu yeniden sertifikasyona alabilirsiniz.
            </p>
          </div>
          <Button nativeButton={false} render={<Link href="/certificates" />}>
            Sertifikalara git
          </Button>
        </section>
      )}

      <section className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="font-heading text-lg">Son oturumlar</h2>
          <Button
            nativeButton={false}
            variant="ghost"
            size="sm"
            render={<Link href="/sessions" />}
          >
            Tümü
            <ChevronRight aria-hidden />
          </Button>
        </div>
        <SessionTable rows={sessionSummaries.slice(0, 4)} />
      </section>
    </div>
  )
}
