"use client"

import * as React from "react"

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
import { AssignmentDialog } from "./assignment-dialog"
import {
  assignmentStatusLabel,
  assignments as initialRows,
  assignableScenarios,
  type Assignment,
} from "./mock"

export default function AssignmentsPage() {
  const [rows, setRows] = React.useState<Assignment[]>(initialRows)

  function assign(input: { employee: string; scenario: string; dueDate: string }) {
    const scenario = assignableScenarios.find(
      (item) => item.name === input.scenario
    )
    setRows((prev) => [
      {
        id: `a-${312 + prev.length}`,
        employee: input.employee,
        scenario: input.scenario,
        version: scenario?.version ?? "v1",
        assignedBy: "Burak Yıldırım",
        assignedAt: new Date().toLocaleDateString("tr-TR"),
        dueDate: input.dueDate,
        status: "pending",
      },
      ...prev,
    ])
  }

  const pendingCount = rows.filter((assignment) => assignment.status === "pending").length
  const overdueCount = rows.filter((assignment) => assignment.status === "overdue").length

  return (
    <div className="space-y-6">
      <PageHeader
        title="Atamalar"
        description="Kim hangi senaryoyu oynayacak. Atama, oynanacak sürümü de sabitler."
        action={<AssignmentDialog onAssign={assign} />}
      />

      <div className="min-w-0 rounded-lg border border-border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[170px]">Çalışan</TableHead>
              <TableHead>Senaryo</TableHead>
              <TableHead className="w-[90px]">Sürüm</TableHead>
              <TableHead className="w-[150px]">Atayan</TableHead>
              <TableHead className="w-[120px]">Son tarih</TableHead>
              <TableHead className="w-[140px]">Durum</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((assignment) => (
              <TableRow key={assignment.id}>
                <TableCell className="font-medium">{assignment.employee}</TableCell>
                <TableCell>
                  {assignment.scenario}
                  {assignment.renewal ? (
                    <div className="prova-meta normal-case">
                      yeniden sertifikasyon
                    </div>
                  ) : null}
                </TableCell>
                <TableCell className="prova-meta">{assignment.version}</TableCell>
                <TableCell className="text-muted-foreground">
                  {assignment.assignedBy}
                </TableCell>
                <TableCell className="prova-meta">{assignment.dueDate}</TableCell>
                <TableCell>
                  <Badge
                    variant="outline"
                    className={
                      assignment.status === "completed" ? "text-muted-foreground" : ""
                    }
                  >
                    {assignmentStatusLabel[assignment.status]}
                  </Badge>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <p className="prova-meta">
        {rows.length} atama · {pendingCount} bekliyor · {overdueCount} gecikti
      </p>
    </div>
  )
}
