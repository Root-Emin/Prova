"use client"

import * as React from "react"

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

  return (
    <div className="space-y-6">
      <PageHeader
        title="Oturumlar"
        description="Oynanmış her sınav ve sonucu. Puanı incelemek ve gerekirse ezmek için satırı açın."
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

      <SessionTable rows={rows} />

      <p className="prova-meta">
        {rows.length} oturum · ham level kaydı hiçbir oturumda saklanmaz
      </p>
    </div>
  )
}
