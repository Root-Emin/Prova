"use client"

import Link from "next/link"
import { ChevronRight } from "lucide-react"

import { cn } from "@/lib/utils"

import { DepartmentDelete } from "./department-delete"
import type { Department, DepartmentStats } from "./mock"

/**
 * Departman kartı. Kartın taşıdığı tek sayı toplam değil dağılımdır: kaç kişi
 * giriş yaptı, kaç kişi bekliyor. Yöneticinin bu ekranda aradığı soru budur.
 *
 * Kartın tamamı departmana giden bağlantıdır; kapatma düğmesi bu bağlantının
 * üstünde ayrı bir katmanda durur, yanlışlıkla tıklanmayacak kadar kenarda.
 */
export function DepartmentCard({
  department,
  stats,
}: {
  department: Department
  stats: DepartmentStats
}) {
  const activeShare = stats.total === 0 ? 0 : (stats.active / stats.total) * 100
  const waitingShare =
    stats.total === 0 ? 0 : (stats.waiting / stats.total) * 100

  return (
    <article
      className={cn(
        "prova-kart group relative flex flex-1 flex-col gap-3 p-4 transition-colors",
        "focus-within:border-foreground hover:border-foreground"
      )}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <span className="prova-meta uppercase">{department.code}</span>
          <h2 className="font-heading text-lg leading-tight">
            <Link
              href={`/users/${department.id}`}
              className="after:absolute after:inset-0 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
            >
              {department.name}
            </Link>
          </h2>
        </div>
        <ChevronRight
          size={16}
          aria-hidden
          className="mt-1 shrink-0 text-muted-foreground group-hover:text-primary"
        />
      </div>

      <p className="prova-meta normal-case">
        {department.lead} · {department.site}
      </p>

      {/* Aktif dolu arduvaz, bekleyen turuncu, pasif nötr yüzey. */}
      <div
        className="flex h-1.5 overflow-hidden rounded-full bg-muted"
        role="presentation"
      >
        <span
          className="bg-primary"
          style={{ width: `${activeShare}%` }}
        />
        <span className="bg-ember" style={{ width: `${waitingShare}%` }} />
      </div>

      <div className="flex items-center justify-between gap-4">
        <dl className="flex items-baseline gap-4">
          <Figure label="Kişi" value={stats.total} strong />
          <Figure label="Aktif" value={stats.active} />
          <Figure label="Bekliyor" value={stats.waiting} />
          {stats.inactive > 0 && <Figure label="Pasif" value={stats.inactive} />}
        </dl>

        <DepartmentDelete
          department={department}
          memberCount={stats.total}
          className="relative z-10 -my-1 -mr-2 text-muted-foreground hover:text-foreground"
        />
      </div>
    </article>
  )
}

function Figure({
  label,
  value,
  strong = false,
}: {
  label: string
  value: number
  strong?: boolean
}) {
  return (
    <div className="flex items-baseline gap-1.5">
      <dt className="prova-meta uppercase">{label}</dt>
      <dd
        className={cn(
          "font-heading leading-none text-primary",
          strong ? "text-xl" : "text-base"
        )}
      >
        {value}
      </dd>
    </div>
  )
}
