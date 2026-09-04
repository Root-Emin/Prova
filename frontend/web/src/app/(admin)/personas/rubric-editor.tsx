"use client"

import { AlertTriangle, Check, Plus, Trash2 } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import type { Criterion } from "./mock"

export function RubricEditor({
  criteria,
  onChange,
  onAdd,
  onSil,
}: {
  criteria: Criterion[]
  onChange: (id: string, patch: Partial<Criterion>) => void
  onAdd: () => void
  onSil: (id: string) => void
}) {
  const totalWeight = criteria.reduce(
    (total, criterion) => total + criterion.weight,
    0
  )
  const mandatoryCount = criteria.filter((criterion) => criterion.mandatory).length
  const weightValid = totalWeight === 100

  return (
    <section className="prova-bolum">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h3 className="text-sm font-medium">Rubrik kriterleri</h3>
          <p className="prova-meta normal-case">
            Zorunlu kriter düşerse sonuç, toplam puandan bağımsız kaldı sayılır.
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={onAdd}>
          <Plus aria-hidden />
          Kriter ekle
        </Button>
      </div>

      {/* The total-weight readout sits above the list. */}
      <div className="mt-3 flex items-center justify-between gap-4 rounded-md border border-border bg-muted px-3 py-2">
        <span className="flex items-center gap-2 text-sm">
          {weightValid ? (
            <Check size={16} className="text-primary" aria-hidden />
          ) : (
            <AlertTriangle size={16} className="text-foreground" aria-hidden />
          )}
          Toplam ağırlık {totalWeight}/100
          {weightValid ? "" : " — 100 olmalı"}
        </span>
        <span className="prova-meta normal-case">
          {criteria.length} kriter · {mandatoryCount} zorunlu
        </span>
      </div>

      <ul className="mt-3 divide-y divide-border border-y border-border">
        {criteria.map((criterion) => (
          <li key={criterion.id} className="flex items-center gap-3 py-2">
            <span className="prova-meta w-[64px] shrink-0 uppercase">
              {criterion.code}
            </span>

            <Input
              value={criterion.name}
              onChange={(event) => onChange(criterion.id, { name: event.target.value })}
              aria-label={`${criterion.code} kriter adı`}
              className="flex-1"
            />

            <Input
              type="number"
              min={0}
              max={100}
              value={criterion.weight}
              onChange={(event) =>
                onChange(criterion.id, { weight: Number(event.target.value) || 0 })
              }
              aria-label={`${criterion.code} ağırlığı`}
              className="w-[72px] shrink-0"
            />

            <label className="flex w-[76px] shrink-0 items-center gap-2">
              <Switch
                checked={criterion.mandatory}
                onCheckedChange={(value) =>
                  onChange(criterion.id, { mandatory: value })
                }
              />
              <span className="prova-meta normal-case">zorunlu</span>
            </label>

            <Button
              variant="ghost"
              size="icon-sm"
              aria-label={`${criterion.code} kriterini sil`}
              onClick={() => onSil(criterion.id)}
            >
              <Trash2 className="size-4" aria-hidden />
            </Button>
          </li>
        ))}
      </ul>
    </section>
  )
}
