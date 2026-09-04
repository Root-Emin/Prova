"use client"

import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Slider } from "@/components/ui/slider"
import { Textarea } from "@/components/ui/textarea"
import { models, type Scenario } from "./mock"

function firmnessLabel(value: number) {
  if (value < 34) return "Uysal — ilk talepte teslim olur"
  if (value < 67) return "Dengeli — gerekçe görürse ikna olur"
  return "İnatçı — gerekçesiz ısrarı reddeder"
}

export function PersonaEditor({
  scenario,
  onChange,
}: {
  scenario: Scenario
  onChange: (patch: Partial<Scenario>) => void
}) {
  return (
    <>
      <section className="prova-bolum">
        <h3 className="text-sm font-medium">Persona</h3>

        {/* Short fields sit in two columns. */}
        <div className="prova-alan-izgara mt-3">
          <div className="space-y-1">
            <Label htmlFor="persona-adi">Persona</Label>
            <Input
              id="persona-adi"
              value={scenario.personaName}
              onChange={(event) => onChange({ personaName: event.target.value })}
            />
          </div>

          <div className="space-y-1">
            <Label>Konuşmayı yürüten model</Label>
            <Select
              items={models}
              value={scenario.model}
              onValueChange={(value) => onChange({ model: value as string })}
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {Object.entries(models).map(([value, label]) => (
                  <SelectItem key={value} value={value}>
                    {label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        <div className="mt-3 space-y-1">
          <Label htmlFor="persona-tanimi">Persona tanımı</Label>
          <Textarea
            id="persona-tanimi"
            rows={3}
            value={scenario.personaDefinition}
            onChange={(event) => onChange({ personaDefinition: event.target.value })}
          />
        </div>

        <div className="mt-4 space-y-2">
          <div className="flex items-baseline justify-between gap-4">
            <Label htmlFor="sertlik">Sertlik seviyesi</Label>
            <span className="prova-meta normal-case">
              {scenario.firmness} · {firmnessLabel(scenario.firmness)}
            </span>
          </div>
          <Slider
            id="sertlik"
            value={scenario.firmness}
            min={0}
            max={100}
            step={1}
            onValueChange={(value) =>
              onChange({ firmness: Array.isArray(value) ? value[0] : value })
            }
          />
        </div>
      </section>

      <section className="prova-bolum">
        <div className="flex items-baseline justify-between gap-4">
          <h3 className="text-sm font-medium">Sistem talimatı</h3>
          <p className="prova-meta normal-case">
            Model bilmeyen tarafı oynar; talimatta doğru cevabı yazma.
          </p>
        </div>
        <Textarea
          id="sistem-talimati"
          rows={6}
          className="mt-2 font-mono text-xs leading-relaxed"
          value={scenario.systemPrompt}
          onChange={(event) => onChange({ systemPrompt: event.target.value })}
        />
      </section>
    </>
  )
}
