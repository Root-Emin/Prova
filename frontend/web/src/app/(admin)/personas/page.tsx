"use client"

import * as React from "react"
import Image from "next/image"
import { Check, ChevronLeft, Save } from "lucide-react"
import { toast } from "sonner"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { PageHeader } from "@/components/prova/page-header"
import { PersonaEditor } from "./persona-editor"
import { PersonaGallery } from "./persona-gallery"
import { RubricEditor } from "./rubric-editor"
import { ScenarioList } from "./scenario-list"
import { VersionPicker } from "./version-picker"
import {
  personaColorClass,
  personaColors,
  type PersonaColor,
} from "./persona-colors"
import {
  scenarios as initialScenarios,
  type Criterion,
  type Scenario,
} from "./mock"

export default function PersonasPage() {
  const [scenarios, setScenarios] = React.useState<Scenario[]>(
    initialScenarios
  )
  /** null iken kart galerisi, dolu iken o rengin ayar ekranı görünür. */
  const [activeColor, setActiveColor] = React.useState<PersonaColor | null>(null)
  const [selectedId, setSelectedId] = React.useState(initialScenarios[0].id)
  const [dirty, setDirty] = React.useState(false)
  const [lastRow, setLastRow] = React.useState<string | null>(null)
  const newCriterionCounter = React.useRef(0)

  const owned = React.useMemo(
    () =>
      activeColor
        ? scenarios.filter((row) => row.color === activeColor)
        : [],
    [scenarios, activeColor]
  )
  const scenario = owned.find((row) => row.id === selectedId) ?? owned[0]

  function openColor(color: PersonaColor) {
    const first = scenarios.find((row) => row.color === color)
    if (first) setSelectedId(first.id)
    setActiveColor(color)
  }

  function closeColor() {
    setActiveColor(null)
  }

  function update(patch: Partial<Scenario>) {
    if (!scenario) return
    setScenarios((prev) =>
      prev.map((row) => (row.id === scenario.id ? { ...row, ...patch } : row))
    )
    setDirty(true)
  }

  function updateCriterion(id: string, patch: Partial<Criterion>) {
    if (!scenario) return
    update({
      criteria: scenario.criteria.map((criterion) =>
        criterion.id === id ? { ...criterion, ...patch } : criterion
      ),
    })
  }

  function addCriterion() {
    if (!scenario) return
    const order = scenario.criteria.length + 1
    const prefix = scenario.criteria[0]?.code.split("-")[0] ?? "KRT"
    update({
      criteria: [
        ...scenario.criteria,
        {
          id: `kr-yeni-${(newCriterionCounter.current += 1)}`,
          code: `${prefix}-${String(order).padStart(2, "0")}`,
          name: "Yeni kriter",
          weight: 0,
          mandatory: false,
        },
      ],
    })
  }

  function removeCriterion(id: string) {
    if (!scenario) return
    update({
      criteria: scenario.criteria.filter((criterion) => criterion.id !== id),
    })
  }

  function newVersion() {
    if (!scenario) return
    const nextNumber = scenario.versions.length + 1
    const created = {
      id: `v-${nextNumber}`,
      label: `v${nextNumber}`,
      date: new Date().toLocaleDateString("tr-TR"),
      published: true,
    }
    update({
      versions: [
        created,
        ...scenario.versions.map((version) => ({ ...version, published: false })),
      ],
      activeVersionId: created.id,
    })
    toast.success(`${created.label} oluşturuldu`, {
      description: "Yeni oturumlar bu sürümle başlar.",
    })
  }

  function save() {
    setDirty(false)
    setLastRow(new Date().toLocaleTimeString("tr-TR"))
    toast.success("Persona kaydedildi", {
      description: "Sonraki oturum bu tanımla başlar.",
    })
  }

  // Galeri: sekme açıldığında dört kişilik rengi karşılar.
  if (!activeColor || !scenario) {
    const publishedCount = scenarios.filter(
      (row) => row.status === "published"
    ).length

    return (
      <div className="space-y-4">
        <PageHeader
          title="Personalar"
          description="Her persona bir kişilik rengidir. Rengi seç, tanımını ve rubriğini kur."
          meta={[
            "4 kişilik rengi",
            `${scenarios.length} senaryo`,
            `${publishedCount} yayında`,
          ]}
        />
        <PersonaGallery scenarios={scenarios} onSec={openColor} />
      </div>
    )
  }

  // Ayar ekranı: seçilen rengin senaryoları, persona tanımı ve rubriği.
  const profile = personaColors[activeColor]
  const renk = personaColorClass[activeColor]
  const live = scenario.versions.find((version) => version.published)

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="sm" onClick={closeColor}>
          <ChevronLeft aria-hidden />
          Tüm personalar
        </Button>
        <span aria-hidden className="h-4 w-px bg-border" />
        <span aria-hidden className={cn("size-2 rounded-full", renk.accent)} />
        <span className="prova-meta normal-case">
          {profile.label} · {profile.title}
        </span>
      </div>

      <PageHeader
        title={`${profile.label} — ${profile.title}`}
        description={profile.summary}
        meta={[
          scenario.versions.find(
            (version) => version.id === scenario.activeVersionId
          )?.label ?? "—",
          scenario.status === "published" ? "yayında" : "taslak",
          scenario.lastEditedBy,
          scenario.lastEditedAt,
          `${scenario.sessionCount} oturum`,
        ]}
        action={
          <VersionPicker
            versions={scenario.versions}
            activeVersionId={scenario.activeVersionId}
            onVersionChange={(id) => update({ activeVersionId: id })}
            onNewVersion={newVersion}
          />
        }
      />

      <div className="flex gap-4">
        <ScenarioList
          scenarios={owned}
          selectedId={scenario.id}
          onSec={(id) => setSelectedId(id)}
        />

        <div className="min-w-0 flex-1">
          {/* The whole editor is one card; sections are split by a thin rule. */}
          <div className="prova-kart">
            <div className="prova-bolum flex gap-4">
              {/* Renk kimliği editörün başında da durur. */}
              <span
                className={cn(
                  "flex h-[100px] w-[84px] shrink-0 items-start overflow-hidden rounded-md border border-border pt-2",
                  renk.soft
                )}
              >
                <Image
                  src={profile.image}
                  alt=""
                  width={96}
                  height={Math.round(
                    (96 * profile.image.height) / profile.image.width
                  )}
                  className="h-auto w-full"
                />
              </span>

              <div className="min-w-0">
                <h2 className="font-heading text-base text-primary">
                  {scenario.name}
                </h2>
                <p className="prova-meta normal-case">{scenario.summary}</p>
                <dl className="mt-2 space-y-1">
                  <div className="flex gap-2">
                    <dt className={cn("prova-meta shrink-0 uppercase", renk.ink)}>
                      Zorluk
                    </dt>
                    <dd className="text-xs leading-snug text-muted-foreground">
                      {profile.challenge}
                    </dd>
                  </div>
                  <div className="flex gap-2">
                    <dt className={cn("prova-meta shrink-0 uppercase", renk.ink)}>
                      Açılış
                    </dt>
                    <dd className="text-xs leading-snug text-muted-foreground">
                      {profile.opening}
                    </dd>
                  </div>
                </dl>
              </div>
            </div>

            <PersonaEditor scenario={scenario} onChange={update} />

            <RubricEditor
              criteria={scenario.criteria}
              onChange={updateCriterion}
              onAdd={addCriterion}
              onSil={removeCriterion}
            />
          </div>

          <div className="prova-kaydet-cubugu mt-3">
            <span className="prova-meta normal-case">
              {dirty
                ? "Kaydedilmemiş değişiklik var"
                : lastRow
                  ? `Kaydedildi ${lastRow}`
                  : `Yayındaki sürüm ${live?.label ?? "—"}`}
            </span>
            <Button onClick={save} disabled={!dirty}>
              {dirty ? <Save aria-hidden /> : <Check aria-hidden />}
              {dirty ? "Kaydet" : "Kayıtlı"}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
