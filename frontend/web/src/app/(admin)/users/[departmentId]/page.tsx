"use client"

import * as React from "react"
import Link from "next/link"
import { notFound, useRouter } from "next/navigation"
import { ArrowLeft, Search } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { PageHeader } from "@/components/prova/page-header"

import { BulkImportDialog } from "../bulk-import-dialog"
import { DepartmentDelete } from "../department-delete"
import { InviteDialog } from "../invite-dialog"
import { statsFor, type UserStatus } from "../mock"
import { UserTable } from "../user-table"
import { useUsers } from "../users-store"

type Filter = "all" | UserStatus

const filterLabel: Record<Filter, string> = {
  all: "Tümü",
  active: "Aktif",
  expected: "Giriş bekleniyor",
  invited: "Davet bekliyor",
  inactive: "Pasif",
}

const filterOrder: Filter[] = ["all", "active", "expected", "invited", "inactive"]

export default function DepartmentPage({
  params,
}: {
  params: Promise<{ departmentId: string }>
}) {
  const { departmentId } = React.use(params)
  const { departments, users } = useUsers()
  const router = useRouter()

  const [filter, setFilter] = React.useState<Filter>("all")
  const [query, setQuery] = React.useState("")
  // Departman bu ekrandan kapatılınca listeye dönülür; dönüş tamamlanana
  // kadar 404 gösterilmez.
  const [closed, setClosed] = React.useState(false)

  const members = React.useMemo(
    () => users.filter((user) => user.departmentId === departmentId),
    [departmentId, users]
  )

  const department = departments.find((item) => item.id === departmentId)
  if (!department) {
    if (closed) return null
    notFound()
  }

  const term = query.trim().toLocaleLowerCase("tr-TR")
  const rows = members.filter((user) => {
    if (filter !== "all" && user.status !== filter) return false
    if (term.length === 0) return true
    return (
      user.name.toLocaleLowerCase("tr-TR").includes(term) ||
      user.email.toLocaleLowerCase("tr-TR").includes(term)
    )
  })

  const stats = statsFor(users, departmentId)

  return (
    <div className="space-y-6">
      <Button
        nativeButton={false}
        variant="ghost"
        size="sm"
        render={<Link href="/users" />}
      >
        <ArrowLeft aria-hidden />
        Kullanıcılar
      </Button>

      <PageHeader
        title={department.name}
        description={`${department.lead} yönetiminde. Kullanıcı masaüstü uygulamasından kendi kurumsal adresiyle giriş yaptığında durumu aktife döner.`}
        meta={[
          department.code,
          department.site,
          `${stats.total} kişi`,
          `${stats.active} giriş yaptı`,
          stats.waiting > 0 && `${stats.waiting} bekliyor`,
        ]}
        action={
          <div className="flex gap-2">
            <DepartmentDelete
              department={department}
              memberCount={stats.total}
              onDeleted={() => {
                setClosed(true)
                router.replace("/users")
              }}
              className="text-muted-foreground hover:text-foreground"
            />
            <BulkImportDialog departmentId={departmentId} lockDepartment />
            <InviteDialog departmentId={departmentId} lockDepartment />
          </div>
        }
      />

      <div className="flex items-center justify-between gap-4">
        <Tabs
          value={filter}
          onValueChange={(value) => setFilter(value as Filter)}
        >
          <TabsList variant="line">
            {filterOrder.map((option) => (
              <TabsTrigger key={option} value={option}>
                {filterLabel[option]}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>

        <div className="relative w-[280px] shrink-0">
          <Search
            size={16}
            aria-hidden
            className="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-muted-foreground"
          />
          <Input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Ad veya e-posta"
            aria-label="Departmanda ara"
            className="pl-8"
          />
        </div>
      </div>

      {rows.length > 0 ? (
        <UserTable rows={rows} />
      ) : (
        <p className="rounded-lg border border-border bg-card px-4 py-8 text-center text-sm text-muted-foreground">
          {members.length === 0
            ? "Bu departmanda kullanıcı yok. Excel ile toplu yükleyebilir veya tek tek davet edebilirsiniz."
            : "Bu ölçütle eşleşen kullanıcı yok."}
        </p>
      )}

      <p className="prova-meta">
        {rows.length} / {members.length} kullanıcı gösteriliyor
      </p>
    </div>
  )
}
