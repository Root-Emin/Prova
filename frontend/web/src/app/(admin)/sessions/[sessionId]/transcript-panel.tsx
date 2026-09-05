"use client"

import { cn } from "@/lib/utils"
import type { TranscriptLine } from "../details"

/** Admin review: the full conversation the score is based on. */
export function TranscriptPanel({
  lines,
}: {
  lines: TranscriptLine[]
}) {
  return (
    <aside className="min-w-0">
      <div className="sticky top-6 overflow-hidden rounded-xl border border-line-strong bg-card">
        <div className="flex items-start justify-between gap-3 border-b border-border px-4 py-4">
          <div>
            <p className="prova-meta uppercase">Kanıt kaydı</p>
            <h2 className="mt-1 font-heading text-xl tracking-tight">Transkript</h2>
            <p className="prova-meta mt-1 normal-case">
              Puanların dayandığı görüşme metni.
            </p>
          </div>
          <span className="prova-meta rounded-md border border-border px-2 py-1 normal-case">
            {lines.length} mesaj
          </span>
        </div>
        <ul className="max-h-[640px] space-y-2 overflow-y-auto bg-muted/30 p-3">
          {lines.map((line) => {
            const isEmployee = line.speaker === "employee"
            return (
              <li
                key={line.id}
                className={cn(
                  "rounded-lg border px-3 py-2.5",
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
