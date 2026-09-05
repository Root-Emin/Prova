"use client"

import Link from "next/link"

import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

import { initialsOf, userRoleLabel, type User } from "./mock"
import { UserStatusBadge } from "./status-badge"
import { UserDelete } from "./user-delete"
import { useUsers } from "./users-store"

/**
 * Departman personeli. Satırın taşıdığı iki sinyal giriş yapılıp yapılmadığı
 * ve hangi makineden yapıldığıdır; sertifikanın dayandığı bağ budur.
 */
export function UserTable({
  rows,
  showDepartment = false,
}: {
  rows: User[]
  /** Arama sonucunda kişinin hangi departmanda olduğu da gösterilir. */
  showDepartment?: boolean
}) {
  const { departments } = useUsers()
  const departmentName = (id: string) =>
    departments.find((department) => department.id === id)?.name ?? "—"

  return (
    <div className="min-w-0 rounded-lg border border-border bg-card">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-[220px]">Kişi</TableHead>
            {showDepartment && (
              <TableHead className="w-[115px]">Departman</TableHead>
            )}
            <TableHead className="w-[120px]">Rol</TableHead>
            <TableHead className="w-[120px]">Durum</TableHead>
            <TableHead className="w-[115px]">Son giriş</TableHead>
            {/* Arama sonucunda departman sütunu yer kaplıyor; makine bilgisi
                departman listesinde ve kişi ekranında zaten duruyor. */}
            {!showDepartment && <TableHead>Bilgisayar</TableHead>}
            <TableHead className="w-[64px] text-right">İşlem</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((user) => {
            const station =
              user.workstations.find((item) => item.current) ??
              user.workstations[0]

            return (
              <TableRow key={user.id}>
                <TableCell>
                  <div className="flex items-center gap-2.5">
                    <Avatar>
                      <AvatarFallback>{initialsOf(user.name)}</AvatarFallback>
                    </Avatar>
                    <div className="min-w-0">
                      <Link
                        href={`/users/${user.departmentId}/${user.id}`}
                        className="block truncate font-medium hover:underline"
                      >
                        {user.name}
                      </Link>
                      <div className="prova-meta truncate normal-case">
                        {user.email}
                      </div>
                    </div>
                  </div>
                </TableCell>

                {showDepartment && (
                  <TableCell className="text-muted-foreground">
                    {departmentName(user.departmentId)}
                  </TableCell>
                )}

                <TableCell>
                  {userRoleLabel[user.role]}
                  <div className="prova-meta normal-case">{user.title}</div>
                </TableCell>

                <TableCell>
                  <UserStatusBadge status={user.status} />
                </TableCell>

                <TableCell className="prova-meta">
                  {user.lastLoginAt ?? "—"}
                </TableCell>

                {!showDepartment && (
                  <TableCell>
                    {station ? (
                      <>
                        <span className="font-mono text-xs">
                          {station.hostname}
                        </span>
                        <div className="prova-meta normal-case">
                          {station.ip}
                        </div>
                      </>
                    ) : (
                      <span className="prova-meta normal-case">
                        Cihaz kaydı yok
                      </span>
                    )}
                  </TableCell>
                )}

                <TableCell className="text-right">
                  <UserDelete
                    user={user}
                    departmentName={departmentName(user.departmentId)}
                    iconOnly
                    className="text-muted-foreground hover:text-foreground"
                  />
                </TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
    </div>
  )
}
