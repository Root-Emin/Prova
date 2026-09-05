"use client"

import * as React from "react"
import { CalendarClock, ClipboardList, Search, Undo2, UserPlus } from "lucide-react"
import { toast } from "sonner"

import { PageHeader } from "@/components/prova/page-header"
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { currentAdmin } from "@/lib/current-admin"

import { SummaryCard } from "../summary-card"
import { today, type User } from "../users/mock"
import { useUsers } from "../users/users-store"
import { AssignmentDialog, type AssignInput } from "./assignment-dialog"
import { UnassignedList } from "./unassigned-list"
import {
  assignments as seed,
  assignableScenarios,
  assignmentStatusLabel,
  dueHint,
  isOpen,
  type AssignableScenario,
  type Assignment,
} from "./mock"

type Filter = "open" | "pending" | "overdue" | "completed" | "all"

const filterLabel: Record<Filter, string> = {
  open: "Açık",
  pending: "Bekliyor",
  overdue: "Gecikti",
  completed: "Tamamlandı",
  all: "Tümü",
}

const filterOrder: Filter[] = ["open", "pending", "overdue", "completed", "all"]

/**
 * Atamalar.
 *
 * Ekran iki soruyu ayrı ayrı cevaplar: "kimin sınavı yok" ve "verilen atamalar
 * ne durumda". Birincisi üstte durur, çünkü yöneticinin bu ekrana gelme sebebi
 * çoğunlukla odur; ikincisi takip listesidir.
 */
export default function AssignmentsPage() {
  const { departments, users } = useUsers()

  const [rows, setRows] = React.useState<Assignment[]>(seed)
  const [selected, setSelected] = React.useState<string[]>([])
  const [targets, setTargets] = React.useState<User[]>([])
  const [dialogOpen, setDialogOpen] = React.useState(false)
  const [filter, setFilter] = React.useState<Filter>("open")
  const [department, setDepartment] = React.useState("all")
  const [query, setQuery] = React.useState("")

  const userById = React.useMemo(
    () => new Map(users.map((user) => [user.id, user])),
    [users]
  )

  const departmentName = React.useCallback(
    (id: string) =>
      departments.find((item) => item.id === id)?.name ?? "Departmansız",
    [departments]
  )

  // Kurumdan çıkarılan kişinin ataması listede durmaz.
  const live = React.useMemo(
    () => rows.filter((row) => userById.has(row.userId)),
    [rows, userById]
  )

  const openUserIds = React.useMemo(
    () => new Set(live.filter(isOpen).map((row) => row.userId)),
    [live]
  )

  /**
   * Sınavı olmayan kişiler: sınava yalnızca çalışanlar girer, pasif hesap
   * oturum açamaz, açık ataması olan da yeniden listeye düşmez.
   */
  const unassigned = React.useMemo(
    () =>
      users.filter(
        (user) =>
          user.role === "employee" &&
          user.status !== "inactive" &&
          !openUserIds.has(user.id)
      ),
    [openUserIds, users]
  )

  const term = query.trim().toLocaleLowerCase("tr-TR")
  const visibleUnassigned = unassigned.filter((user) => {
    if (department !== "all" && user.departmentId !== department) return false
    if (term.length === 0) return true
    return (
      user.name.toLocaleLowerCase("tr-TR").includes(term) ||
      user.email.toLocaleLowerCase("tr-TR").includes(term)
    )
  })

  const departmentOptions = React.useMemo(
    () => ({
      all: "Tüm departmanlar",
      ...Object.fromEntries(
        departments.map((item) => [item.id, item.name])
      ),
    }),
    [departments]
  )

  const hintFor = React.useCallback(
    (user: User) => {
      const done = live.filter(
        (row) => row.userId === user.id && row.status === "completed"
      )
      if (done.length === 0) return ""
      return `son sınav ${done[0].dueDate}`
    },
    [live]
  )

  const listedRows = live.filter((row) => {
    if (filter === "all") return true
    if (filter === "open") return isOpen(row)
    return row.status === filter
  })

  const pendingCount = live.filter((row) => row.status === "pending").length
  const overdueCount = live.filter((row) => row.status === "overdue").length
  const completedCount = live.filter((row) => row.status === "completed").length

  function toggle(userId: string) {
    setSelected((previous) =>
      previous.includes(userId)
        ? previous.filter((id) => id !== userId)
        : [...previous, userId]
    )
  }

  function openDialog(people: User[]) {
    setTargets(people)
    setDialogOpen(true)
  }

  /** Her kişi–senaryo çifti ayrı bir atama satırıdır; takip satır bazında yapılır. */
  function assign({ userIds, scenarioIds, dueDate }: AssignInput) {
    const picked = scenarioIds
      .map((id) => assignableScenarios.find((item) => item.id === id))
      .filter((item): item is AssignableScenario => Boolean(item))
    if (picked.length === 0) return
    const stamp = today()

    setRows((previous) => {
      let sequence = 0
      const created = userIds.flatMap((userId) =>
        picked.map((scenario) => ({
          id: `a-${400 + previous.length + sequence++}`,
          userId,
          scenarioId: scenario.id,
          scenario: scenario.name,
          version: scenario.version,
          assignedBy: currentAdmin.name,
          assignedAt: stamp,
          dueDate,
          status: "pending" as const,
        }))
      )
      return [...created, ...previous]
    })
    setSelected([])
  }

  function withdraw(assignment: Assignment) {
    setRows((previous) => previous.filter((row) => row.id !== assignment.id))
    setSelected([])
    toast.success("Atama geri alındı", {
      description: `${userById.get(assignment.userId)?.name ?? "Çalışan"} · ${assignment.scenario}`,
    })
  }

  const allVisibleSelected =
    visibleUnassigned.length > 0 &&
    visibleUnassigned.every((user) => selected.includes(user.id))

  return (
    <div className="space-y-6">
      <PageHeader
        title="Atamalar"
        description="Kim hangi senaryoyu oynayacak. Atama, oynanacak sürümü de sabitler."
        action={
          <Button size="lg" onClick={() => openDialog([])}>
            <UserPlus aria-hidden />
            Sınav ata
          </Button>
        }
      />

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <SummaryCard
          title="Sınavı olmayan"
          value={String(unassigned.length)}
          description="Açık ataması bulunmayan çalışan"
          icon={UserPlus}
        />
        <SummaryCard
          title="Bekliyor"
          value={String(pendingCount)}
          description="Atandı, henüz oynanmadı"
          icon={ClipboardList}
        />
        <SummaryCard
          title="Gecikti"
          value={String(overdueCount)}
          description="Son tarihi geçti"
          icon={CalendarClock}
        />
        <SummaryCard
          title="Tamamlandı"
          value={String(completedCount)}
          description="Oturumu oynandı ve puanlandı"
          icon={ClipboardList}
        />
      </div>

      <section className="space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 className="font-heading text-lg">
            Sınavı olmayan kişiler
            <span className="prova-meta ml-2 normal-case">
              {visibleUnassigned.length} kişi
            </span>
          </h2>

          <div className="flex items-center gap-2">
            <Select
              items={departmentOptions}
              value={department}
              onValueChange={(value) => setDepartment(value as string)}
            >
              <SelectTrigger className="w-[190px] bg-card">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Tüm departmanlar</SelectItem>
                {departments.map((item) => (
                  <SelectItem key={item.id} value={item.id}>
                    {item.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <div className="relative w-[220px]">
              <Search
                size={16}
                aria-hidden
                className="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-muted-foreground"
              />
              <Input
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder="Ad veya e-posta"
                aria-label="Sınavı olmayan kişilerde ara"
                className="bg-card pl-8"
              />
            </div>
          </div>
        </div>

        {visibleUnassigned.length > 0 ? (
          <>
            <label className="flex w-fit cursor-pointer items-center gap-2.5 text-sm">
              <Checkbox
                checked={allVisibleSelected}
                onCheckedChange={() =>
                  setSelected(
                    allVisibleSelected
                      ? []
                      : visibleUnassigned.map((user) => user.id)
                  )
                }
              />
              Listedeki herkesi seç
            </label>

            <UnassignedList
              people={visibleUnassigned}
              selected={selected}
              departmentName={departmentName}
              hintFor={hintFor}
              onToggle={toggle}
              onAssign={(user) => openDialog([user])}
            />
          </>
        ) : (
          <p className="rounded-lg border border-border bg-card px-4 py-8 text-center text-sm text-muted-foreground">
            {unassigned.length === 0
              ? "Sınavı olmayan çalışan kalmadı; herkesin açık bir ataması var."
              : "Bu ölçütle eşleşen kişi yok."}
          </p>
        )}

        {selected.length > 0 && (
          <div className="prova-kaydet-cubugu">
            <span className="text-sm">
              {selected.length} kişi seçildi
            </span>
            <div className="flex items-center gap-2">
              <Button variant="ghost" onClick={() => setSelected([])}>
                Seçimi bırak
              </Button>
              <Button
                onClick={() =>
                  openDialog(
                    selected
                      .map((id) => userById.get(id))
                      .filter((user): user is User => Boolean(user))
                  )
                }
              >
                Seçilenlere sınav ata
              </Button>
            </div>
          </div>
        )}
      </section>

      <section className="space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 className="font-heading text-lg">Verilen atamalar</h2>
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
        </div>

        {listedRows.length > 0 ? (
          <div className="min-w-0 rounded-lg border border-border bg-card">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[170px]">Çalışan</TableHead>
                  <TableHead>Senaryo</TableHead>
                  <TableHead className="w-[120px]">Atayan</TableHead>
                  <TableHead className="w-[130px]">Son tarih</TableHead>
                  <TableHead className="w-[110px]">Durum</TableHead>
                  <TableHead className="w-[104px] text-right">İşlem</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {listedRows.map((assignment) => {
                  const person = userById.get(assignment.userId)
                  const hint = dueHint(assignment)
                  return (
                    <TableRow key={assignment.id}>
                      <TableCell className="font-medium">
                        {person?.name ?? "—"}
                        <div className="prova-meta normal-case">
                          {person ? departmentName(person.departmentId) : ""}
                        </div>
                      </TableCell>
                      <TableCell>
                        {assignment.scenario}
                        <div className="prova-meta normal-case">
                          sürüm {assignment.version}
                          {assignment.renewal
                            ? " · yeniden sertifikasyon"
                            : ""}
                        </div>
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {assignment.assignedBy}
                      </TableCell>
                      <TableCell className="prova-meta">
                        {assignment.dueDate}
                        {hint ? (
                          <div className="prova-meta normal-case">{hint}</div>
                        ) : null}
                      </TableCell>
                      <TableCell>
                        <Badge
                          variant="outline"
                          className={
                            assignment.status === "completed"
                              ? "text-muted-foreground"
                              : assignment.status === "overdue"
                                ? "border-line-strong"
                                : ""
                          }
                        >
                          {assignmentStatusLabel[assignment.status]}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-right">
                        {isOpen(assignment) ? (
                          <WithdrawAssignment
                            assignment={assignment}
                            employeeName={person?.name ?? "Çalışan"}
                            onWithdraw={() => withdraw(assignment)}
                          />
                        ) : (
                          <span className="prova-meta normal-case">—</span>
                        )}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </div>
        ) : (
          <p className="rounded-lg border border-border bg-card px-4 py-8 text-center text-sm text-muted-foreground">
            Bu ölçütle eşleşen atama yok.
          </p>
        )}

        <p className="prova-meta">
          {listedRows.length} / {live.length} atama gösteriliyor
        </p>
      </section>

      <AssignmentDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        targets={targets}
        candidates={unassigned}
        departmentName={departmentName}
        onAssign={assign}
      />
    </div>
  )
}

/** Oynanmamış atama geri alınabilir; oynanmış oturum kaydı silinmez. */
function WithdrawAssignment({
  assignment,
  employeeName,
  onWithdraw,
}: {
  assignment: Assignment
  employeeName: string
  onWithdraw: () => void
}) {
  const [open, setOpen] = React.useState(false)

  return (
    <AlertDialog open={open} onOpenChange={setOpen}>
      <AlertDialogTrigger render={<Button variant="ghost" size="sm" />}>
        <Undo2 aria-hidden />
        Geri al
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Atama geri alınsın mı?</AlertDialogTitle>
          <AlertDialogDescription>
            {employeeName} için “{assignment.scenario}” sınavı masaüstü
            uygulamasından kaldırılır. Kişi yeniden sınavı olmayanlar listesine
            döner.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Vazgeç</AlertDialogCancel>
          <Button
            onClick={() => {
              setOpen(false)
              onWithdraw()
            }}
          >
            Atamayı geri al
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
