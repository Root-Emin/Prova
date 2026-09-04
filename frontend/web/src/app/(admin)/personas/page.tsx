"use client"

import * as React from "react"
import { Check, Save } from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { PageHeader } from "@/components/prova/page-header"
import { PersonaEditor } from "./persona-editor"
import { RubricEditor } from "./rubric-editor"
import { ScenarioList } from "./scenario-list"
import { VersionPicker } from "./version-picker"
import {
  scenarios as initialScenarios,
  type Criterion,
  type Scenario,
} from "./mock"

export default function PersonasPage() {
  const [scenarios, setScenarios] = React.useState<Scenario[]>(
    initialScenarios
  )
  const [selectedId, setSelectedId] = React.useState(initialScenarios[0].id)
  const [dirty, setDirty] = React.useState(false)
  const [lastRow, setLastRow] = React.useState<string | null>(null)
  const newCriterionCounter = React.useRef(0)

  const scenario = scenarios.find((row) => row.id === selectedId)!

  function update(patch: Partial<Scenario>) {
    setScenarios((prev) =>
      prev.map((row) =>
        row.id === selectedId ? { ...row, ...patch } : row
      )
    )
    setDirty(true)
  }

  function updateCriterion(id: string, patch: Partial<Criterion>) {
    update({
      criteria: scenario.criteria.map((criterion) =>
        criterion.id === id ? { ...criterion, ...patch } : criterion
      ),
    })
  }

  function addCriterion() {
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
    update({
      criteria: scenario.criteria.filter((criterion) => criterion.id !== id),
    })
  }

  function newVersion() {
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

  const live = scenario.versions.find((version) => version.published)

  return (
    <div className="space-y-4">
      <PageHeader
        title="Personalar"
        description="Karakterin tanımı, sertliği ve rubriği burada kurulur."
        meta={[
          scenario.versions.find((version) => version.id === scenario.activeVersionId)
            ?.label ?? "—",
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
          scenarios={scenarios}
          selectedId={selectedId}
          onSec={(id) => setSelectedId(id)}
        />

        <div className="min-w-0 flex-1">
          {/* The whole editor is one card; sections are split by a thin rule. */}
          <div className="prova-kart">
            <div className="prova-bolum">
              <h2 className="font-heading text-base text-primary">
                {scenario.name}
              </h2>
              <p className="prova-meta normal-case">{scenario.summary}</p>
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
