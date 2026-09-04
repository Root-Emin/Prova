"use client"

import Link from "next/link"
import { Award, CalendarClock, ChevronRight, Play, RefreshCw } from "lucide-react"

import { Button } from "@/components/ui/button"
import { RubricStatusInline } from "@/components/prova/rubric-status"
import { TopBar } from "@/components/top-bar"
import { useDeviceState } from "@/hooks/use-device-state"
import { assignedSessions, pastSessions } from "./mock"

export default function MySessionsPage() {
  const device = useDeviceState()

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <TopBar />

      <main className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
        <div className="mx-auto w-[720px]">
          <h1 className="font-heading text-2xl tracking-tight">Oturumlarım</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Sınav yalnızca kayıtlı cihazdan verilebilir. Bu cihaz{" "}
            {device.registered ? "kayıtlı" : "kayıtlı değil"}.
          </p>

          <section className="mt-6 space-y-3">
            <h2 className="text-sm font-medium">Sana atanan</h2>

            {assignedSessions.map((assignment) => (
              <article
                key={assignment.id}
                className="flex items-center justify-between gap-6 rounded-lg border border-line-strong bg-card px-4 py-3"
              >
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <h3 className="font-heading text-base">{assignment.scenario}</h3>
                    {assignment.renewal ? (
                      <span className="inline-flex items-center gap-1 rounded-md border border-border bg-muted px-1.5 py-0.5">
                        <RefreshCw size={16} className="text-primary" aria-hidden />
                        <span className="prova-meta">Yenileme</span>
                      </span>
                    ) : null}
                  </div>
                  <p className="prova-meta normal-case">
                    {assignment.persona} · sürüm {assignment.version} · atayan {assignment.assignedBy}
                  </p>
                  <p className="prova-meta mt-1 inline-flex items-center gap-1.5 normal-case">
                    <CalendarClock size={16} aria-hidden />
                    Son tarih {assignment.dueDate}
                  </p>
                </div>

                <Button
                  nativeButton={false}
                  size="lg"
                  className="shrink-0"
                  render={<Link href="/session-briefing" />}
                >
                  <Play aria-hidden />
                  Başla
                </Button>
              </article>
            ))}
          </section>

          <section className="mt-8 space-y-3">
            <h2 className="text-sm font-medium">Geçmiş oturumların</h2>

            <ul className="divide-y divide-border rounded-lg border border-border bg-card">
              {pastSessions.map((session) => (
                <li
                  key={session.id}
                  className="flex items-center justify-between gap-6 px-4 py-3"
                >
                  <div className="min-w-0">
                    <p className="text-sm">{session.scenario}</p>
                    <p className="prova-meta normal-case">
                      {session.date} · sürüm {session.version} · {session.score} puan
                      {session.certificateNo ? (
                        <>
                          {" · "}
                          <Award size={16} className="inline align-text-bottom" aria-hidden />{" "}
                          {session.certificateNo}
                        </>
                      ) : null}
                    </p>
                  </div>

                  <div className="flex shrink-0 items-center gap-4">
                    <RubricStatusInline status={session.result} />
                    <Button
                      nativeButton={false}
                      variant="ghost"
                      size="sm"
                      render={<Link href="/result" />}
                    >
                      Sonucu gör
                      <ChevronRight aria-hidden />
                    </Button>
                  </div>
                </li>
              ))}
            </ul>
          </section>
        </div>
      </main>
    </div>
  )
}
