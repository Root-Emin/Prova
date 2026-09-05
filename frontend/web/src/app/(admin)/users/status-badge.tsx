import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"

import { userStatusLabel, type UserStatus } from "./mock"

/**
 * Dört durum, dört renk yok. Ayrım dolgu ağırlığındadır: aktif dolu arduvaz,
 * bekleyenler kalın çerçeve, pasif sakin metin. Rozet içinde ikon yok — rozet
 * primitifi ikonu 12px'e indiriyor, tasarım kuralı 16/20 dışına çıkmıyor.
 */
const statusClass: Record<UserStatus, string> = {
  active: "",
  expected: "border-line-strong",
  invited: "border-line-strong",
  inactive: "text-muted-foreground",
}

export function UserStatusBadge({
  status,
  className,
}: {
  status: UserStatus
  className?: string
}) {
  return (
    <Badge
      variant={status === "active" ? "default" : "outline"}
      className={cn(statusClass[status], className)}
    >
      {userStatusLabel[status]}
    </Badge>
  )
}
