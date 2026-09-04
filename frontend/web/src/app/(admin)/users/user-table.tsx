"use client"

import { Ban, RotateCcw } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  userStatusLabel,
  userRoleLabel,
  type User,
} from "./mock"

export function UserTable({
  rows,
  onStatusChange,
}: {
  rows: User[]
  onStatusChange: (id: string) => void
}) {
  return (
    <div className="rounded-lg border border-border bg-card">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-[240px]">Ad soyad</TableHead>
            <TableHead>E-posta</TableHead>
            <TableHead className="w-[160px]">Rol</TableHead>
            <TableHead className="w-[140px]">Durum</TableHead>
            <TableHead className="w-[140px] text-right">İşlem</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((user) => (
            <TableRow key={user.id}>
              <TableCell className="font-medium">
                {user.name}
                <div className="prova-meta">{user.id}</div>
              </TableCell>
              <TableCell className="text-muted-foreground">
                {user.email}
              </TableCell>
              <TableCell>{userRoleLabel[user.role]}</TableCell>
              <TableCell>
                <Badge
                  variant="outline"
                  className={
                    user.status === "inactive" ? "text-muted-foreground" : ""
                  }
                >
                  {userStatusLabel[user.status]}
                </Badge>
              </TableCell>
              <TableCell className="text-right">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onStatusChange(user.id)}
                >
                  {user.status === "inactive" ? (
                    <>
                      <RotateCcw aria-hidden />
                      Etkinleştir
                    </>
                  ) : (
                    <>
                      <Ban aria-hidden />
                      Pasifleştir
                    </>
                  )}
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
