"use client"

import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"

import { initialsOf, type User } from "../users/mock"
import { UserStatusBadge } from "../users/status-badge"

/**
 * Sınavı olmayan kişiler.
 *
 * Atama ekranının asıl işi burada yapılır: yönetici "kime sınav vereceğim"
 * sorusunu tabloda arayarak değil, hazır süzülmüş bu listede cevaplar. Satır
 * tek kişiye atamayı, işaret kutuları toplu atamayı açar.
 */
export function UnassignedList({
  people,
  selected,
  departmentName,
  hintFor,
  onToggle,
  onAssign,
}: {
  people: User[]
  selected: string[]
  departmentName: (departmentId: string) => string
  /** Kişinin sınav geçmişi: bir satırda okunacak kadar kısa. */
  hintFor: (user: User) => string
  onToggle: (userId: string) => void
  onAssign: (user: User) => void
}) {
  return (
    <ul className="min-w-0 divide-y divide-border rounded-lg border border-border bg-card">
      {people.map((user) => {
        const checked = selected.includes(user.id)
        return (
          <li
            key={user.id}
            className="flex items-center gap-3 px-3 py-2.5 sm:px-4"
          >
            <Checkbox
              checked={checked}
              onCheckedChange={() => onToggle(user.id)}
              aria-label={`${user.name} kişisini seç`}
            />

            <Avatar className="hidden shrink-0 sm:flex">
              <AvatarFallback>{initialsOf(user.name)}</AvatarFallback>
            </Avatar>

            <div className="min-w-0 flex-1">
              <div className="truncate text-sm font-medium">{user.name}</div>
              <div className="prova-meta truncate normal-case">
                {departmentName(user.departmentId)} · {user.title}
              </div>
            </div>

            <div className="hidden w-[190px] shrink-0 lg:block">
              <p className="prova-meta truncate normal-case">
                {hintFor(user)}
              </p>
            </div>

            <div className="hidden w-[140px] shrink-0 md:block">
              <UserStatusBadge status={user.status} />
            </div>

            <Button
              variant="outline"
              size="sm"
              className="shrink-0"
              onClick={() => onAssign(user)}
            >
              Sınav ata
            </Button>
          </li>
        )
      })}
    </ul>
  )
}
