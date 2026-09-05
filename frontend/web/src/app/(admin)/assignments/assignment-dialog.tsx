"use client"

import * as React from "react"
import { CalendarDays, Check, Search, X } from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { cn } from "@/lib/utils"

import {
  personaColorClass,
  personaColorOrder,
  personaColors,
  type PersonaColor,
} from "../personas/persona-colors"
import type { User } from "../users/mock"
import { assignableScenarios, formatDate } from "./mock"

export type AssignInput = {
  userIds: string[]
  /** Bir atamada birden çok sınav verilebilir; her senaryo ayrı atama olur. */
  scenarioIds: string[]
  dueDate: string
}

/** Son tarih hazır aralıklarla verilir; takvimle uğraşmak çoğu atamada gereksiz. */
const presets = [
  { days: 7, label: "1 hafta" },
  { days: 14, label: "2 hafta" },
  { days: 30, label: "1 ay" },
]

function isoAfter(days: number) {
  const date = new Date()
  date.setDate(date.getDate() + days)
  return date.toISOString().slice(0, 10)
}

/**
 * Sınav atama.
 *
 * Diyalog iki yoldan açılır: listeden kişi seçilerek (kişiler hazır gelir) ya
 * da başlıktaki düğmeyle (kişiler burada seçilir). İkisinde de sıra aynıdır:
 * kimler, hangi kişilik renkleri, o renklerin hangi senaryoları, hangi tarihe
 * kadar. Renk adımı senaryo listesini kısaltmak için değil, atamanın kararı
 * olduğu için önde durur: yönetici "çalışan hangi tip müşteriyle sınansın"
 * sorusunu senaryo adlarına bakmadan cevaplar.
 */
export function AssignmentDialog({
  open,
  onOpenChange,
  targets,
  candidates,
  departmentName,
  onAssign,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Listeden seçilerek gelen kişiler; boşsa diyalog kendi seçicisini açar. */
  targets: User[]
  /** Sınav atanabilecek kişiler. */
  candidates: User[]
  departmentName: (departmentId: string) => string
  onAssign: (input: AssignInput) => void
}) {
  const [chosen, setChosen] = React.useState<string[]>([])
  const [colors, setColors] = React.useState<PersonaColor[]>([])
  const [scenarioIds, setScenarioIds] = React.useState<string[]>([])
  const [dueDate, setDueDate] = React.useState(() => isoAfter(7))
  const [query, setQuery] = React.useState("")
  const [picking, setPicking] = React.useState(false)

  // Diyalog her açılışta seçili kişilerle baştan kurulur. Sıfırlama render
  // sırasında yapılır; efekt kullanmak burada fazladan bir tur render demek.
  const [wasOpen, setWasOpen] = React.useState(open)
  if (open !== wasOpen) {
    setWasOpen(open)
    if (open) {
      setChosen(targets.map((user) => user.id))
      setPicking(targets.length === 0)
      setColors([])
      setScenarioIds([])
      setDueDate(isoAfter(7))
      setQuery("")
    }
  }

  const people = React.useMemo(() => {
    const byId = new Map<string, User>()
    for (const user of [...targets, ...candidates]) byId.set(user.id, user)
    return byId
  }, [candidates, targets])

  const term = query.trim().toLocaleLowerCase("tr-TR")
  const listed = candidates.filter(
    (user) =>
      term.length === 0 ||
      user.name.toLocaleLowerCase("tr-TR").includes(term) ||
      user.email.toLocaleLowerCase("tr-TR").includes(term)
  )

  // Seçili renklerin senaryoları renk sırasıyla listelenir; sıra galerideki
  // kart sırasıyla aynı olsun ki yönetici aynı düzeni iki ekranda da bulsun.
  const offered = React.useMemo(
    () =>
      personaColorOrder
        .filter((color) => colors.includes(color))
        .flatMap((color) =>
          assignableScenarios.filter((item) => item.color === color)
        ),
    [colors]
  )

  const picked = assignableScenarios.filter((item) =>
    scenarioIds.includes(item.id)
  )
  const ready = chosen.length > 0 && picked.length > 0 && dueDate !== ""

  function toggle(userId: string) {
    setChosen((previous) =>
      previous.includes(userId)
        ? previous.filter((id) => id !== userId)
        : [...previous, userId]
    )
  }

  /** Renk bırakılınca o renkten seçilmiş sınavlar da listede kalmaz. */
  function toggleColor(color: PersonaColor) {
    if (!colors.includes(color)) {
      setColors([...colors, color])
      return
    }
    setColors(colors.filter((item) => item !== color))
    setScenarioIds((previous) =>
      previous.filter(
        (id) =>
          assignableScenarios.find((item) => item.id === id)?.color !== color
      )
    )
  }

  function toggleScenario(scenarioId: string) {
    setScenarioIds((previous) =>
      previous.includes(scenarioId)
        ? previous.filter((id) => id !== scenarioId)
        : [...previous, scenarioId]
    )
  }

  function submit(event: React.FormEvent) {
    event.preventDefault()
    if (!ready) return
    const due = formatDate(new Date(`${dueDate}T00:00:00`))
    onAssign({ userIds: chosen, scenarioIds: picked.map((item) => item.id), dueDate: due })

    const exam = picked.length === 1 ? "sınav" : `${picked.length} sınav`
    toast.success(
      chosen.length === 1
        ? `${people.get(chosen[0])?.name ?? "Kişi"} için ${exam} atandı`
        : `${chosen.length} kişiye ${exam} atandı`,
      {
        description: `${picked
          .map((item) => `${item.name} · ${item.version}`)
          .join(" | ")} · son tarih ${due}`,
      }
    )
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[560px]">
        <form onSubmit={submit}>
          <DialogHeader>
            <DialogTitle>Sınav ata</DialogTitle>
            <DialogDescription>
              Atanan senaryolar çalışanın masaüstü uygulamasında görünür ve
              atama anında yayında olan sürümle oynanır.
            </DialogDescription>
          </DialogHeader>

          <div className="max-h-[62vh] space-y-5 overflow-y-auto py-5">
            <section className="space-y-2">
              <div className="flex items-baseline justify-between gap-3">
                <Label>Kimlere</Label>
                <span className="prova-meta normal-case">
                  {chosen.length} kişi seçili
                </span>
              </div>

              {chosen.length > 0 && (
                <ul className="flex flex-wrap gap-1.5">
                  {chosen.map((id) => {
                    const person = people.get(id)
                    if (!person) return null
                    return (
                      <li key={id}>
                        <button
                          type="button"
                          onClick={() => toggle(id)}
                          aria-label={`${person.name} kişisini seçimden çıkar`}
                          className="flex items-center gap-1.5 rounded-full border border-border bg-muted py-1 pr-1.5 pl-2.5 text-sm hover:border-line-strong"
                        >
                          {person.name}
                          <X size={16} aria-hidden className="text-muted-foreground" />
                        </button>
                      </li>
                    )
                  })}
                </ul>
              )}

              {picking ? (
                <PersonPicker
                  chosen={chosen}
                  departmentName={departmentName}
                  listed={listed}
                  onToggle={toggle}
                  onQueryChange={setQuery}
                  query={query}
                />
              ) : (
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => setPicking(true)}
                >
                  Kişi ekle
                </Button>
              )}
            </section>

            <section className="space-y-2">
              <div className="flex items-baseline justify-between gap-3">
                <Label>Kişilik rengi</Label>
                <span className="prova-meta normal-case">
                  birden çok renk seçilebilir
                </span>
              </div>
              <ColorPicker chosen={colors} onToggle={toggleColor} />
            </section>

            <section className="space-y-2">
              <div className="flex items-baseline justify-between gap-3">
                <Label>Sınavlar</Label>
                <span className="prova-meta normal-case">
                  {picked.length} sınav seçili
                </span>
              </div>

              {colors.length === 0 ? (
                <p className="rounded-lg border border-dashed border-border px-3 py-6 text-center text-sm text-muted-foreground">
                  Önce kişilik rengi seçin; o renklerin yayındaki senaryoları
                  burada listelenir.
                </p>
              ) : (
                <ul
                  aria-label="Atanacak sınavlar"
                  className="grid max-h-[220px] gap-2 overflow-y-auto"
                >
                  {offered.map((item) => {
                    const active = scenarioIds.includes(item.id)
                    const renk = personaColorClass[item.color]
                    return (
                      <li key={item.id}>
                        <button
                          type="button"
                          role="checkbox"
                          aria-checked={active}
                          onClick={() => toggleScenario(item.id)}
                          className={cn(
                            "flex w-full items-start gap-2.5 rounded-lg border px-3 py-2.5 text-left transition-colors",
                            active
                              ? cn(renk.border, renk.soft)
                              : "border-border hover:border-line-strong"
                          )}
                        >
                          <span
                            aria-hidden
                            className={cn(
                              "mt-1 size-2 shrink-0 rounded-full",
                              renk.accent
                            )}
                          />
                          <span className="min-w-0 flex-1">
                            <span className="flex items-baseline justify-between gap-3">
                              <span className="text-sm font-medium">
                                {item.name}
                              </span>
                              <span className="prova-meta">{item.version}</span>
                            </span>
                            <span className="prova-meta mt-0.5 block normal-case">
                              {item.persona} · {item.sector}
                            </span>
                          </span>
                        </button>
                      </li>
                    )
                  })}
                </ul>
              )}
            </section>

            <section className="space-y-2">
              <Label htmlFor="atama-son-tarih">Son tarih</Label>
              <div className="flex flex-wrap items-center gap-2">
                <div className="relative">
                  <CalendarDays
                    size={16}
                    aria-hidden
                    className="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-muted-foreground"
                  />
                  <Input
                    id="atama-son-tarih"
                    type="date"
                    value={dueDate}
                    min={new Date().toISOString().slice(0, 10)}
                    onChange={(event) => setDueDate(event.target.value)}
                    className="w-[190px] pl-8"
                    required
                  />
                </div>
                {presets.map((preset) => (
                  <Button
                    key={preset.days}
                    type="button"
                    variant={dueDate === isoAfter(preset.days) ? "default" : "outline"}
                    size="sm"
                    onClick={() => setDueDate(isoAfter(preset.days))}
                  >
                    {preset.label}
                  </Button>
                ))}
              </div>
            </section>
          </div>

          <DialogFooter>
            <DialogClose render={<Button variant="outline" type="button" />}>
              Vazgeç
            </DialogClose>
            <Button type="submit" disabled={!ready}>
              {submitLabel(chosen.length, picked.length)}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

/** Düğme kaç atama açılacağını yazar; çarpım tablodaki satır sayısıdır. */
function submitLabel(peopleCount: number, examCount: number) {
  if (peopleCount === 0 || examCount === 0) return "Ata"
  if (peopleCount === 1) {
    return examCount === 1 ? "Ata" : `${examCount} sınav ata`
  }
  return examCount === 1
    ? `${peopleCount} kişiye ata`
    : `${peopleCount} kişiye ${examCount} sınav ata`
}

/**
 * Kişilik rengi seçimi: personalar galerisindeki dört kartın diyaloğa sığan
 * hâli. Yayında senaryosu olmayan renk atanamaz, o yüzden kapalı gösterilir.
 */
function ColorPicker({
  chosen,
  onToggle,
}: {
  chosen: PersonaColor[]
  onToggle: (color: PersonaColor) => void
}) {
  return (
    <ul aria-label="Kişilik renkleri" className="grid grid-cols-4 gap-2">
      {personaColorOrder.map((color) => {
        const profile = personaColors[color]
        const renk = personaColorClass[color]
        const count = assignableScenarios.filter(
          (item) => item.color === color
        ).length
        const active = chosen.includes(color)
        const empty = count === 0

        return (
          <li key={color} className="flex">
            <button
              type="button"
              role="checkbox"
              aria-checked={active}
              disabled={empty}
              onClick={() => onToggle(color)}
              aria-label={`${profile.label} — ${profile.title}`}
              className={cn(
                "flex flex-1 flex-col overflow-hidden rounded-lg border text-left transition-colors",
                active
                  ? cn(renk.border, renk.soft)
                  : "border-border hover:border-line-strong",
                empty && "cursor-not-allowed opacity-50 hover:border-border"
              )}
            >
              {/* Rengin kimliği kartın üstünde ince bant olarak durur. */}
              <span className={cn("h-1 w-full shrink-0", renk.accent)} />

              <span className="flex flex-1 flex-col gap-0.5 px-2.5 py-2">
                <span className="flex items-center justify-between gap-1">
                  <span className="prova-meta uppercase">{profile.label}</span>
                  {active && (
                    <Check size={14} aria-hidden className={renk.ink} />
                  )}
                </span>
                <span
                  className={cn(
                    "font-heading text-sm leading-tight",
                    active ? renk.ink : "text-foreground"
                  )}
                >
                  {profile.title}
                </span>
                <span className="prova-meta mt-auto normal-case">
                  {empty ? "yayında yok" : `${count} senaryo`}
                </span>
              </span>
            </button>
          </li>
        )
      })}
    </ul>
  )
}

/** Diyalog içinden kişi seçimi: arama ve işaret kutuları. */
function PersonPicker({
  chosen,
  departmentName,
  listed,
  onToggle,
  onQueryChange,
  query,
}: {
  chosen: string[]
  departmentName: (departmentId: string) => string
  listed: User[]
  onToggle: (userId: string) => void
  onQueryChange: (query: string) => void
  query: string
}) {
  return (
    <div className="space-y-2">
      <div className="relative">
        <Search
          size={16}
          aria-hidden
          className="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-muted-foreground"
        />
        <Input
          value={query}
          onChange={(event) => onQueryChange(event.target.value)}
          placeholder="Ad veya e-posta ile ara"
          aria-label="Atanacak kişiyi ara"
          className="pl-8"
        />
      </div>

      {listed.length > 0 ? (
        <ul className="max-h-[200px] divide-y divide-border overflow-y-auto rounded-lg border border-border">
          {listed.map((user) => (
            <li key={user.id}>
              <label className="flex cursor-pointer items-center gap-3 px-3 py-2 hover:bg-muted">
                <Checkbox
                  checked={chosen.includes(user.id)}
                  onCheckedChange={() => onToggle(user.id)}
                />
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm font-medium">
                    {user.name}
                  </span>
                  <span className="prova-meta block truncate normal-case">
                    {departmentName(user.departmentId)} · {user.email}
                  </span>
                </span>
              </label>
            </li>
          ))}
        </ul>
      ) : (
        <p className="rounded-lg border border-border px-3 py-4 text-center text-sm text-muted-foreground">
          Sınav atanabilecek kişi bulunamadı.
        </p>
      )}
    </div>
  )
}
