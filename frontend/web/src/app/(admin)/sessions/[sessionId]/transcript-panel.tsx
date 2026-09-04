"use client"

import { cn } from "@/lib/utils"
import type { TranscriptLine } from "../details"

/** Trainer review: the full conversation the score is based on. */
export function TranscriptPanel({
  lines,
}: {
  lines: TranscriptLine[]
}) {
  return (
    <aside className="w-[300px] shrink-0">
      <div className="sticky top-9 rounded-lg border border-border bg-card">
        <div className="border-b border-border px-4 py-3">
          <h2 className="text-sm font-medium">Transkript</h2>
          <p className="prova-meta normal-case">
            Metin cihazda üretildi, ham ses saklanmadı.
          </p>
        </div>
        <ul className="max-h-[560px] space-y-2 overflow-y-auto p-3">
          {lines.map((line) => {
            const isEmployee = line.speaker === "employee"
            return (
              <li
                key={line.id}
                className={cn(
                  "rounded-md border px-3 py-2",
                  isEmployee
                    ? "ml-4 border-line-strong bg-muted"
                    : "mr-4 border-border bg-background"
                )}
              >
                <div className="prova-meta flex items-center justify-between uppercase">
                  <span>{line.name}</span>
                  <span>{line.time}</span>
                </div>
                <p className="mt-1 text-sm leading-relaxed">{line.text}</p>
              </li>
            )
          })}
        </ul>
      </div>
    </aside>
  )
}
