"use client"

import * as React from "react"

import { cn } from "@/lib/utils"
import type { TranscriptLine } from "@/app/session/mock"

/** The transcript is the only scrollable region on the page. */
export function Transcript({
  lines,
  employeeName,
  characterName,
}: {
  lines: TranscriptLine[]
  employeeName: string
  characterName: string
}) {
  const sonRef = React.useRef<HTMLDivElement>(null)

  React.useEffect(() => {
    sonRef.current?.scrollIntoView({ block: "end" })
  }, [lines.length])

  return (
    <div className="min-h-0 flex-1 overflow-y-auto rounded-lg border border-border bg-background p-4">
      <ul className="space-y-3">
        {lines.map((line) => {
          const isEmployee = line.speaker === "employee"
          return (
            <li
              key={line.id}
              className={cn(
                "rounded-md border px-3 py-2",
                isEmployee
                  ? "ml-10 border-line-strong bg-muted"
                  : "mr-10 border-border bg-card"
              )}
            >
              <div className="prova-meta flex items-center justify-between uppercase">
                <span>{isEmployee ? `${employeeName} · siz` : `${characterName} · karakter`}</span>
                <span>{line.time}</span>
              </div>
              <p className="mt-1 text-sm leading-relaxed text-foreground">
                {line.text}
              </p>
            </li>
          )
        })}
      </ul>
      <div ref={sonRef} />
    </div>
  )
}
