"use client"

import * as React from "react"
import { CheckCircle2, Clock3, ListChecks, TrendingUp } from "lucide-react"

import { PageHeader } from "@/components/prova/page-header"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { SessionTable } from "./session-table"
import { sessionSummaries } from "./mock"

const tabs = [
  { value: "all", label: "Tümü" },
  { value: "failed", label: "Kaldı" },
  { value: "passed", label: "Geçti" },
  { value: "evaluating", label: "Değerlendiriliyor" },
] as const

export default function SessionsPage() {
  const [tab, setTab] =
    React.useState<(typeof tabs)[number]["value"]>("all")

  const rows = sessionSummaries.filter((session) => {
    if (tab === "all") return true
    if (tab === "evaluating") return session.status === "evaluating"
    return session.status === "evaluated" && session.result === tab
  })
  const evaluatedSessions = sessionSummaries.filter(
    (session) => session.status === "evaluated"
  )
  const passedSessions = evaluatedSessions.filter(
    (session) => session.result === "passed"
  )
  const averageScore = evaluatedSessions.length
    ? Math.round(
        evaluatedSessions.reduce((total, session) => total + (session.score ?? 0), 0) /
          evaluatedSessions.length
      )
    : 0

  return (
    <div className="space-y-6">
      <PageHeader
        title="Oturumlar"
        description="Çalışanların tamamladığı sınavları ve değerlendirme sonuçlarını buradan inceleyin."
      />

      <Tabs value={tab} onValueChange={(value) => setTab(value as typeof tab)}>
        <div className="min-w-0 overflow-x-auto">
          <TabsList>
            {tabs.map((item) => (
              <TabsTrigger key={item.value} value={item.value}>
                {item.label}
              </TabsTrigger>
            ))}
          </TabsList>
        </div>
      </Tabs>

      <section className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4" aria-label="Oturum özeti">
        {[
          {
            label: "Toplam oturum",
            value: sessionSummaries.length,
            note: "tüm kayıtlar",
            icon: ListChecks,
          },
          {
            label: "Değerlendirildi",
            value: evaluatedSessions.length,
            note: `${sessionSummaries.length - evaluatedSessions.length} bekliyor`,
            icon: CheckCircle2,
          },
          {
            label: "Başarı oranı",
            value: evaluatedSessions.length
              ? `${Math.round((passedSessions.length / evaluatedSessions.length) * 100)}%`
              : "—",
            note: `${passedSessions.length} geçti`,
            icon: TrendingUp,
          },
          {
            label: "Ortalama puan",
            value: evaluatedSessions.length ? averageScore : "—",
            note: "değerlendirilenler",
            icon: Clock3,
          },
        ].map((stat) => (
          <div
            key={stat.label}
            className="rounded-lg border border-border bg-card px-4 py-3"
          >
            <div className="flex items-center justify-between gap-3">
              <span className="prova-meta normal-case">{stat.label}</span>
              <stat.icon size={16} className="text-muted-foreground" aria-hidden />
            </div>
            <p className="mt-2 font-heading text-2xl leading-none text-primary">
              {stat.value}
            </p>
            <p className="prova-meta mt-1 normal-case">{stat.note}</p>
          </div>
        ))}
      </section>

      <SessionTable rows={rows} />

      <p className="prova-meta">
        {rows.length} oturum · ham level kaydı hiçbir oturumda saklanmaz
      </p>
    </div>
  )
}
