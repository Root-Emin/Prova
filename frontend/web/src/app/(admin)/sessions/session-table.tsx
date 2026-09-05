"use client"

import Link from "next/link"
import { ChevronRight, Scale } from "lucide-react"

import { RubricStatusInline } from "@/components/prova/rubric-status"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import type { SessionSummary } from "./mock"

export function SessionTable({ rows }: { rows: SessionSummary[] }) {
  return (
    <div className="min-w-0 rounded-lg border border-border bg-card">
      <Table className="table-fixed !min-w-0">
        <TableHeader>
          <TableRow>
            <TableHead className="w-[82px]">Oturum</TableHead>
            <TableHead className="w-[120px]">Çalışan</TableHead>
            <TableHead>Senaryo</TableHead>
            <TableHead className="w-[126px]">Tarih</TableHead>
            <TableHead className="w-[64px]">Süre</TableHead>
            <TableHead className="w-[54px]">Puan</TableHead>
            <TableHead className="w-[150px]">Sonuç</TableHead>
            <TableHead className="w-[76px]" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((session) => {
            const openable = session.status === "evaluated"
            return (
              <TableRow key={session.id}>
                <TableCell className="font-mono text-xs">{session.id}</TableCell>
                <TableCell className="font-medium">{session.employee}</TableCell>
                <TableCell className="whitespace-normal">
                  {session.scenario}
                  <div className="prova-meta normal-case">
                    sürüm {session.version} · {session.device}
                  </div>
                </TableCell>
                <TableCell className="prova-meta">{session.date}</TableCell>
                <TableCell className="prova-meta font-mono">
                  {session.duration}
                </TableCell>
                <TableCell className="font-medium">
                  {openable ? session.score : "—"}
                </TableCell>
                <TableCell className="whitespace-normal">
                  {openable && session.result ? (
                    <span className="flex items-center gap-2">
                      <RubricStatusInline status={session.result} />
                      {session.mandatoryFailed ? (
                        <span className="prova-meta normal-case">
                          zorunlu düştü
                        </span>
                      ) : null}
                      {session.overridden ? (
                        <Scale
                          size={16}
                          className="text-muted-foreground"
                          aria-label="Puan ezildi"
                        />
                      ) : null}
                    </span>
                  ) : (
                    <span className="prova-meta normal-case">
                      Güçlü model değerlendiriyor
                    </span>
                  )}
                </TableCell>
                <TableCell className="whitespace-normal">
                  {openable ? (
                    <Link
                      href={`/sessions/${session.id}`}
                      aria-label={`${session.id} oturumunu incele`}
                      className="inline-flex h-7 items-center gap-1 rounded-md border border-border px-2.5 text-xs font-medium text-foreground transition-colors hover:bg-muted"
                    >
                      İncele
                      <ChevronRight size={16} aria-hidden />
                    </Link>
                  ) : null}
                </TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
    </div>
  )
}
