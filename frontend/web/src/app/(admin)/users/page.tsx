"use client"

import * as React from "react"
import { Building2, Search, UserCheck, Users as UsersIcon, UserPlus } from "lucide-react"

import { Input } from "@/components/ui/input"
import { PageHeader } from "@/components/prova/page-header"

import { SummaryCard } from "../summary-card"
import { BulkImportDialog } from "./bulk-import-dialog"
import { DepartmentCard } from "./department-card"
import { DepartmentDialog } from "./department-dialog"
import { InviteDialog } from "./invite-dialog"
import { statsFor } from "./mock"
import { UserTable } from "./user-table"
import { useUsers } from "./users-store"

/**
 * Kullanıcılar sekmesinin girişi departman listesidir; kişiler bir departmanın
 * içinde durur. Arama kutusu dolduğunda görünüm kişi listesine geçer, çünkü
 * "şu kişi hangi departmanda" sorusu departmanlar arasında dolaşmadan
 * cevaplanabilmeli.
 */
export default function UsersPage() {
  const { departments, users } = useUsers()
  const [query, setQuery] = React.useState("")

  const term = query.trim().toLocaleLowerCase("tr-TR")
  const matches = React.useMemo(() => {
    if (term.length === 0) return []
    return users.filter(
      (user) =>
        user.name.toLocaleLowerCase("tr-TR").includes(term) ||
        user.email.toLocaleLowerCase("tr-TR").includes(term)
    )
  }, [term, users])

  const firstDepartmentId = departments[0]?.id ?? ""
  const active = users.filter((user) => user.status === "active").length
  const waiting = users.filter(
    (user) => user.status === "expected" || user.status === "invited"
  ).length

  return (
    <div className="space-y-6">
      <PageHeader
        title="Kullanıcılar"
        description="Kurum departmanlara ayrılmıştır. Bir departmana girerek personelini, giriş durumunu ve kayıtlı cihazını görebilirsiniz."
        action={
          <div className="flex gap-2">
            <DepartmentDialog />
            <BulkImportDialog departmentId={firstDepartmentId} />
            <InviteDialog departmentId={firstDepartmentId} />
          </div>
        }
      />

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <SummaryCard
          title="Departman"
          value={String(departments.length)}
          description="Her kullanıcı tek departmana bağlıdır"
          icon={Building2}
        />
        <SummaryCard
          title="Kullanıcı"
          value={String(users.length)}
          description="Kurum alanındaki toplam kayıt"
          icon={UsersIcon}
        />
        <SummaryCard
          title="Giriş yaptı"
          value={String(active)}
          description="Masaüstünden en az bir kez bağlandı"
          icon={UserCheck}
        />
        <SummaryCard
          title="Giriş bekleniyor"
          value={String(waiting)}
          description="Adres tanımlı, ilk giriş yapılmadı"
          icon={UserPlus}
        />
      </div>

      <div className="relative">
        <Search
          size={16}
          aria-hidden
          className="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-muted-foreground"
        />
        <Input
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="Tüm departmanlarda kişi ara — ad veya e-posta"
          aria-label="Kişi ara"
          className="pl-8"
        />
      </div>

      {term.length > 0 ? (
        <section className="space-y-3">
          <h2 className="font-heading text-lg">
            Arama sonucu
            <span className="prova-meta ml-2 normal-case">
              {matches.length} kişi
            </span>
          </h2>
          {matches.length > 0 ? (
            <UserTable rows={matches} showDepartment />
          ) : (
            <p className="rounded-lg border border-border bg-card px-4 py-6 text-center text-sm text-muted-foreground">
              “{query}” ile eşleşen kullanıcı yok.
            </p>
          )}
        </section>
      ) : (
        <section className="space-y-3">
          <h2 className="font-heading text-lg">Departmanlar</h2>
          {departments.length > 0 ? (
            <ul className="grid grid-cols-1 gap-4 lg:grid-cols-2">
              {departments.map((department) => (
                <li key={department.id} className="flex">
                  <DepartmentCard
                    department={department}
                    stats={statsFor(users, department.id)}
                  />
                </li>
              ))}
            </ul>
          ) : (
            <p className="rounded-lg border border-border bg-card px-4 py-8 text-center text-sm text-muted-foreground">
              Henüz departman yok. Personel tanımlamadan önce bir departman
              açın.
            </p>
          )}
        </section>
      )}
    </div>
  )
}
