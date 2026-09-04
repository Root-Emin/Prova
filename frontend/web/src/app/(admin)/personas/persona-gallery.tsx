"use client"

import Image from "next/image"

import { cn } from "@/lib/utils"
import type { Scenario } from "./mock"
import {
  personaColorClass,
  personaColorOrder,
  personaColors,
  type PersonaColor,
} from "./persona-colors"

/**
 * Personalar sekmesinin giriş görünümü: dört kişilik rengi, dört kart.
 * Karta basmak o rengin ayar ekranını açar.
 */
export function PersonaGallery({
  scenarios,
  onSec,
}: {
  scenarios: Scenario[]
  onSec: (color: PersonaColor) => void
}) {
  return (
    <ul className="grid grid-cols-4 gap-4">
      {personaColorOrder.map((color) => {
        const profile = personaColors[color]
        const renk = personaColorClass[color]
        const owned = scenarios.filter((scenario) => scenario.color === color)
        const sessionCount = owned.reduce(
          (total, scenario) => total + scenario.sessionCount,
          0
        )

        return (
          <li key={color} className="flex">
            <button
              type="button"
              onClick={() => onSec(color)}
              aria-label={`${profile.label} — ${profile.title} personasını aç`}
              className={cn(
                "prova-kart group flex flex-1 flex-col overflow-hidden text-left transition-colors",
                "focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring",
                renk.hoverBorder
              )}
            >
              {/* Rengin kimliği kartın üstünde ince bant olarak durur. */}
              <span className={cn("h-1 w-full shrink-0", renk.accent)} />

              {/* İllüstrasyonlar saydam zeminli; panel rengin açık tonudur. */}
              <span
                className={cn(
                  "flex h-[210px] shrink-0 items-start overflow-hidden border-b border-border",
                  renk.soft
                )}
              >
                <Image
                  src={profile.image}
                  alt=""
                  sizes="240px"
                  className="h-auto w-full"
                />
              </span>

              <span className="flex flex-1 flex-col gap-2 p-3">
                <span className="flex items-baseline justify-between gap-2">
                  <span className="prova-meta uppercase">{profile.label}</span>
                  <span className="prova-meta normal-case">
                    {owned.length} senaryo
                  </span>
                </span>

                <span className="block">
                  <span
                    className={cn(
                      "font-heading block text-lg leading-tight",
                      renk.ink
                    )}
                  >
                    {profile.title}
                  </span>
                  <span className="mt-1 block text-sm leading-snug text-foreground">
                    {profile.summary}
                  </span>
                </span>

                {/* Rengin özellikleri — kartın alt yarısı. */}
                <span className="mt-1 flex flex-1 flex-col gap-1.5 border-t border-border pt-2">
                  {profile.traits.map((trait) => (
                    <span key={trait} className="flex gap-2">
                      <span
                        aria-hidden
                        className={cn("mt-1.5 size-1 shrink-0", renk.accent)}
                      />
                      <span className="text-xs leading-snug text-muted-foreground">
                        {trait}
                      </span>
                    </span>
                  ))}
                </span>

                <span className="prova-meta mt-1 block border-t border-border pt-2 normal-case">
                  {sessionCount} oturum
                </span>
              </span>
            </button>
          </li>
        )
      })}
    </ul>
  )
}
