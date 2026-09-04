"use client"

import * as React from "react"
import Link from "next/link"
import { ChevronRight } from "lucide-react"

import { PageHeader } from "@/components/prova/page-header"
import { Badge } from "@/components/ui/badge"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Recertification } from "./recertification"
import {
  certificateStatusLabel,
  certificates as initialRows,
  type Certificate,
} from "./mock"

export default function CertificatesPage() {
  const [rows, setRows] = React.useState<Certificate[]>(initialRows)

  const affected = rows.filter(
    (certificate) => certificate.status === "recertification"
  )

  function startRenewal() {
    setRows((prev) =>
      prev.map((certificate) =>
        certificate.status === "recertification"
          ? { ...certificate, status: "expiring" }
          : certificate
      )
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Sertifikalar"
        description="Geçen her oturum bir sertifika kaydı üretir. Kayıt, senaryo ve rubrik sürümünü, tarihi ve cihazı taşır."
      />

      <Recertification
        affected={affected}
        onStart={startRenewal}
      />

      <div className="rounded-lg border border-border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[160px]">Sertifika no</TableHead>
              <TableHead className="w-[170px]">Çalışan</TableHead>
              <TableHead>Senaryo</TableHead>
              <TableHead className="w-[100px]">Rubrik</TableHead>
              <TableHead className="w-[130px]">Geçerlilik</TableHead>
              <TableHead className="w-[230px]">Durum</TableHead>
              <TableHead className="w-[40px]" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((certificate) => (
              <TableRow key={certificate.id}>
                <TableCell className="font-mono text-xs">
                  {certificate.id}
                </TableCell>
                <TableCell className="font-medium">
                  {certificate.employee}
                </TableCell>
                <TableCell>
                  {certificate.scenario}
                  <div className="prova-meta normal-case">
                    {certificate.score} puan · {certificate.device}
                  </div>
                </TableCell>
                <TableCell className="prova-meta">
                  {certificate.rubricVersion}
                </TableCell>
                <TableCell className="prova-meta">
                  {certificate.validUntil}
                </TableCell>
                <TableCell>
                  <Badge
                    variant="outline"
                    className={
                      certificate.status === "expired"
                        ? "text-muted-foreground"
                        : certificate.status === "recertification"
                          ? "border-line-strong"
                          : ""
                    }
                  >
                    {certificateStatusLabel[certificate.status]}
                  </Badge>
                </TableCell>
                <TableCell>
                  <Link
                    href={`/certificates/${certificate.id}`}
                    aria-label={`${certificate.id} sertifikasını aç`}
                    className="inline-flex text-muted-foreground hover:text-primary"
                  >
                    <ChevronRight size={16} aria-hidden />
                  </Link>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <p className="prova-meta">
        {rows.length} sertifika kaydı · hesap silinse dahi kimliksizleştirilmiş
        biçimde 5 yıl saklanır
      </p>
    </div>
  )
}
