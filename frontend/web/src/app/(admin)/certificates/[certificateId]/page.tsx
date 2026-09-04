"use client"

import * as React from "react"
import Link from "next/link"
import { notFound } from "next/navigation"
import { ArrowLeft, Printer } from "lucide-react"

import { Button } from "@/components/ui/button"
import { ResultView } from "@/components/prova/result/result-view"
import { sessionDetails } from "../../sessions/details"
import { certificateStatusLabel, certificates } from "../mock"

export default function CertificatePage({
  params,
}: {
  params: Promise<{ certificateId: string }>
}) {
  const { certificateId } = React.use(params)
  const certificate = certificates.find((row) => row.id === certificateId)
  if (!certificate) notFound()

  const detail = sessionDetails[certificate.sessionId]

  return (
    <div className="prova-yazdirilabilir space-y-6">
      <div className="prova-yazdirma-disi flex items-center justify-between">
        <Button
          nativeButton={false}
          variant="ghost"
          size="sm"
          render={<Link href="/certificates" />}
        >
          <ArrowLeft aria-hidden />
          Sertifikalar
        </Button>
        <Button variant="outline" onClick={() => window.print()}>
          <Printer aria-hidden />
          Yazdır
        </Button>
      </div>

      <section className="rounded-lg border border-line-strong bg-card px-5 py-4">
        <div className="prova-yazdirma-basligi mb-3">
          <span className="font-heading text-lg text-primary">Prova</span>
          <span className="prova-meta ml-2 uppercase">
            Anadolu Katılım Bankası · yetkinlik sertifikası
          </span>
        </div>

        <div className="flex items-start justify-between gap-8">
          <div>
            <p className="prova-meta uppercase">Sertifika no</p>
            <p className="font-mono text-base">{certificate.id}</p>
            <h1 className="mt-3 font-heading text-xl tracking-tight">
              {certificate.employee}
            </h1>
            <p className="text-sm text-muted-foreground">{certificate.scenario}</p>
          </div>

          <dl className="grid grid-cols-2 gap-x-8 gap-y-2 text-sm">
            <dt className="prova-meta uppercase">Rubrik sürümü</dt>
            <dd>{certificate.rubricVersion}</dd>
            <dt className="prova-meta uppercase">Veriliş</dt>
            <dd>{certificate.issuedAt}</dd>
            <dt className="prova-meta uppercase">Geçerlilik sonu</dt>
            <dd>{certificate.validUntil}</dd>
            <dt className="prova-meta uppercase">Sınav cihazı</dt>
            <dd className="font-mono text-xs">{certificate.device}</dd>
            <dt className="prova-meta uppercase">Oturum</dt>
            <dd className="font-mono text-xs">{certificate.sessionId}</dd>
            <dt className="prova-meta uppercase">Durum</dt>
            <dd>{certificateStatusLabel[certificate.status]}</dd>
          </dl>
        </div>
      </section>

      {detail ? (
        <ResultView
          info={detail.info}
          criteria={detail.criteria}
          canOverride={false}
        />
      ) : (
        <p className="prova-meta normal-case">
          Bu sertifikanın dayandığı oturum kaydı arşive taşınmış.
        </p>
      )}
    </div>
  )
}
