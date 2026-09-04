"use client"

import Link from "next/link"
import { ArrowLeft, Mic } from "lucide-react"

import { Button } from "@/components/ui/button"
import { MicCheck } from "@/components/session/mic-check"
import { PrivacyNotice } from "@/components/prova/privacy-notice"
import { RubricStatusInline } from "@/components/prova/rubric-status"
import { TopBar } from "@/components/top-bar"
import { sessionCriteria } from "@/app/session/mock"
import { sessionBriefing } from "@/app/mock"

export default function SessionBriefingPage() {
  return (
    <div className="flex h-full flex-col overflow-hidden">
      <TopBar />

      <main className="prova-texture flex min-h-0 flex-1 items-center justify-center overflow-y-auto px-6 py-8">
        <div className="w-[560px]">
          <Button
            nativeButton={false}
            variant="ghost"
            size="sm"
            className="-ml-2.5 mb-2"
            render={<Link href="/" />}
          >
            <ArrowLeft aria-hidden />
            Oturumlarım
          </Button>
          <p className="prova-meta uppercase">Atanmış oturum</p>
          <h1 className="mt-1 font-heading text-2xl tracking-tight">
            {sessionBriefing.scenario}
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Karakter: {sessionBriefing.persona} · sürüm{" "}
            {sessionBriefing.version} · {sessionBriefing.assignedBy}
          </p>

          <PrivacyNotice className="mt-5" />

          <section className="mt-5 rounded-lg border border-line-strong bg-card">
            <div className="flex items-center justify-between border-b border-border px-4 py-3">
              <h2 className="text-sm font-medium">
                Ölçülecek kriterler
              </h2>
              <span className="prova-meta">
                Geçme eşiği {sessionBriefing.passThreshold} · süre{" "}
                {sessionBriefing.estimatedDuration}
              </span>
            </div>
            <ul className="divide-y divide-border">
              {sessionCriteria.map((criterion) => (
                <li
                  key={criterion.id}
                  className="flex items-center justify-between gap-4 px-4 py-2.5"
                >
                  <span className="flex min-w-0 items-center gap-3">
                    <span className="prova-meta w-[68px] shrink-0 uppercase">
                      {criterion.code}
                    </span>
                    <span className="truncate text-sm">{criterion.name}</span>
                  </span>
                  <RubricStatusInline status="unevaluated" />
                </li>
              ))}
            </ul>
          </section>

          <MicCheck className="mt-5" />

          <div className="mt-5 flex items-center justify-between">
            <p className="prova-meta normal-case">
              Konuşmak için space tuşunu basılı tutacaksınız.
            </p>
            <Button nativeButton={false} size="lg" render={<Link href="/session" />}>
              <Mic aria-hidden />
              Sınavı başlat
            </Button>
          </div>
        </div>
      </main>
    </div>
  )
}
