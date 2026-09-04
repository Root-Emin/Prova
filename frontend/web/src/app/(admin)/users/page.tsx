"use client"

import * as React from "react"
import { toast } from "sonner"

import { PageHeader } from "@/components/prova/page-header"
import { InviteDialog } from "./invite-dialog"
import { UserTable } from "./user-table"
import { users as initialRows, type User } from "./mock"

export default function UsersPage() {
  const [rows, setRows] =
    React.useState<User[]>(initialRows)

  function toggleStatus(id: string) {
    setRows((prev) =>
      prev.map((user) => {
        if (user.id !== id) return user
        const newStatus = user.status === "inactive" ? "active" : "inactive"
        toast.success(
          newStatus === "inactive"
            ? `${user.name} pasifleştirildi`
            : `${user.name} yeniden etkinleştirildi`
        )
        return { ...user, status: newStatus }
      })
    )
  }

  function invite(created: {
    name: string
    email: string
    role: User["role"]
  }) {
    setRows((prev) => [
      {
        id: `k-${1048 + prev.length}`,
        name: created.name,
        email: created.email,
        role: created.role,
        status: "invite-pending",
        addedAt: new Date().toLocaleDateString("tr-TR"),
      },
      ...prev,
    ])
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Kullanıcılar"
        description="Kurumdaki çalışan, eğitmen ve yöneticiler. Pasifleştirilen kullanıcı oturum açamaz."
        action={<InviteDialog onInvite={invite} />}
      />
      <UserTable rows={rows} onStatusChange={toggleStatus} />
      <p className="prova-meta">
        {rows.length} kullanıcı · Son güncelleme 03.09.2026
      </p>
    </div>
  )
}
