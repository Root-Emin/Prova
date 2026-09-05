"use client"

import Link from "next/link"
import {
  Award,
  ChevronRight,
  ClipboardList,
  Clock3,
  KeyRound,
  Laptop,
  MailCheck,
  MessagesSquare,
  Percent,
  Scale,
  ShieldAlert,
  UserCheck,
  Users,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import { PageHeader } from "@/components/prova/page-header"
import { assignments } from "./assignments/mock"
import { auditEntries } from "./audit-log/mock"
import { certificates, needsRenewal } from "./certificates/mock"
import { scenarios } from "./personas/mock"
import { personaColorOrder } from "./personas/persona-colors"
import { sessionSummaries } from "./sessions/mock"
import { SessionTable } from "./sessions/session-table"
import { useUsers } from "./users/users-store"
import { dashboardSummary, otpSentLastDay } from "./mock"
import { AuditDigest } from "./audit-digest"
import { CoverageCard } from "./coverage-card"
import { PersonaShareCard } from "./persona-share"
import { PriorityActions } from "./priority-actions"
import { SummaryCard } from "./summary-card"
import { TrendChart } from "./trend-chart"
import { VerificationCard } from "./verification-card"

export default function DashboardPage() {
  // Kişi listesi panelin tamamında paylaşılır; sayaçlar da o listeden okunur,
  // böylece bir kullanıcı silindiğinde panel ile kullanıcı ekranı ayrışmaz.
  const { users } = useUsers()

  const employees = users.filter((user) => user.role === "employee")
  const signedIn = users.filter((user) => user.status === "active")
  const waiting = users.filter(
    (user) => user.status === "expected" || user.status === "invited"
  )
  const workstations = users.flatMap((user) => user.workstations)
  const verifiedRate = users.length
    ? Math.round((signedIn.length / users.length) * 100)
    : 0

  const toRenew = needsRenewal()
  const validCertificates = certificates.filter(
    (certificate) =>
      certificate.status === "valid" || certificate.status === "expiring"
  )
  const staleCertificates = certificates.filter(
    (certificate) =>
      certificate.status === "recertification" ||
      certificate.status === "expired"
  )
  const certifiedEmployees = new Set(
    certificates.map((certificate) => certificate.employee)
  )

  const pendingList = assignments.filter(
    (assignment) => assignment.status !== "completed"
  )
  const overdueList = assignments.filter(
    (assignment) => assignment.status === "overdue"
  )

  const evaluating = sessionSummaries.filter(
    (session) => session.status === "evaluating"
  )
  const mandatoryFailed = sessionSummaries.filter(
    (session) => session.mandatoryFailed
  )
  const overridden = sessionSummaries.filter((session) => session.overridden)
  const rejectedLogins = auditEntries.filter((entry) =>
    entry.action.includes("reddedildi")
  )

  const personaShare = personaColorOrder.map((color) => ({
    color,
    sessions: scenarios
      .filter((scenario) => scenario.color === color)
      .reduce((total, scenario) => total + scenario.sessionCount, 0),
  }))

  return (
    <div className="space-y-6">
      <PageHeader
        title="Panel"
        description="Kurumun sözlü yetkinlik, sertifikasyon ve güvenlik görünümü."
        action={
          <span className="prova-meta flex items-center gap-2 normal-case">
            <Clock3 size={16} aria-hidden />
            Son güncelleme {dashboardSummary.updatedAt}
          </span>
        }
      />

      <section
        className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3"
        aria-label="Kurum sayaçları"
      >
        <SummaryCard
          title="Haftalık oturum"
          value={String(dashboardSummary.sessionsThisWeek)}
          description={`${evaluating.length} oturum değerlendiriliyor`}
          icon={MessagesSquare}
        />
        <SummaryCard
          title="Geçme oranı"
          value={dashboardSummary.passRate}
          description="Zorunlu kriter düşenler dahil"
          icon={Percent}
          critical
        />
        <SummaryCard
          title="Bekleyen atama"
          value={String(pendingList.length)}
          description={
            overdueList.length > 0
              ? `${overdueList.length} atama son tarihi geçti`
              : "Tamamı süresi içinde"
          }
          icon={ClipboardList}
          critical={overdueList.length > 0}
        />
        <SummaryCard
          title="Yenilenecek sertifika"
          value={String(toRenew.length)}
          description="Rubrik sürümü eskidi"
          icon={Award}
          critical={toRenew.length > 0}
        />
        <SummaryCard
          title="Kayıtlı cihaz"
          value={String(workstations.length)}
          description="Sınav yalnızca bu makinelerden verilir"
          icon={Laptop}
        />
        <SummaryCard
          title="Doğrulanmış kullanıcı"
          value={`%${verifiedRate}`}
          description={`${waiting.length} hesap ilk girişini bekliyor`}
          icon={UserCheck}
        />
      </section>

      {toRenew.length > 0 && (
        <section className="flex flex-col items-stretch justify-between gap-4 rounded-lg border border-line-strong bg-tint px-4 py-3 sm:flex-row sm:items-center sm:gap-6">
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

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-5">
        <TrendChart />
        <PriorityActions
          items={[
            {
              title: "Yeniden sertifikasyon",
              detail: "Eski rubrikle verilmiş sertifika",
              count: toRenew.length,
              href: "/certificates",
              icon: Award,
              critical: true,
            },
            {
              title: "Zorunlu kriter düşen oturum",
              detail: "Toplam puandan bağımsız kaldı",
              count: mandatoryFailed.length,
              href: "/sessions",
              icon: MessagesSquare,
            },
            {
              title: "Gecikmiş atama",
              detail: "Son tarihi geçti, oturum açılmadı",
              count: overdueList.length,
              href: "/assignments",
              icon: ClipboardList,
            },
            {
              title: "İlk girişini yapmamış hesap",
              detail: "Davet edildi, adres doğrulanmadı",
              count: waiting.length,
              href: "/users",
              icon: Users,
            },
            {
              title: "Puan ezme kaydı",
              detail: "Yönetici gerekçesiyle değiştirildi",
              count: overridden.length,
              href: "/audit-log",
              icon: Scale,
            },
          ]}
        />
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <CoverageCard
          rows={[
            {
              label: "Sertifikalı çalışan",
              note: `${certifiedEmployees.size} / ${employees.length} çalışan`,
              value: certifiedEmployees.size,
              total: employees.length,
            },
            {
              label: "Geçerli sertifika",
              note: `${validCertificates.length} / ${certificates.length} sertifika`,
              value: validCertificates.length,
              total: certificates.length,
            },
            {
              label: "Yenilenmesi gereken sertifika",
              note: `${staleCertificates.length} / ${certificates.length} sertifika`,
              value: staleCertificates.length,
              total: certificates.length,
            },
          ]}
        />
        <VerificationCard
          rows={[
            {
              label: "Adresini doğrulamış hesap",
              value: `${signedIn.length} / ${users.length}`,
              icon: MailCheck,
            },
            {
              label: "Son 24 saatte gönderilen kod",
              value: String(otpSentLastDay),
              icon: KeyRound,
            },
            {
              label: "Kayıtlı çalışma makinesi",
              value: String(workstations.length),
              icon: Laptop,
            },
            {
              label: "Reddedilen giriş denemesi",
              value: String(rejectedLogins.length),
              icon: ShieldAlert,
            },
          ]}
        />
        <PersonaShareCard rows={personaShare} />
      </div>

      <section className="space-y-3">
        <div className="flex items-center justify-between gap-4">
          <div>
            <h2 className="font-heading text-base tracking-tight text-primary">
              Son oturumlar
            </h2>
            <p className="prova-meta mt-1 normal-case">
              Gerçekleşen son görüşme simülasyonları
            </p>
          </div>
          <Button
            nativeButton={false}
            variant="ghost"
            size="sm"
            className="shrink-0"
            render={<Link href="/sessions" />}
          >
            Tümü
            <ChevronRight aria-hidden />
          </Button>
        </div>
        <SessionTable rows={sessionSummaries.slice(0, 4)} />
      </section>

      <AuditDigest rows={auditEntries.slice(0, 5)} />
    </div>
  )
}
