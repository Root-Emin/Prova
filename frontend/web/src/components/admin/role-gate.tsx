"use client"

import { usePathname } from "next/navigation"
import { Lock } from "lucide-react"

import { useRole } from "@/components/admin/role-context"
import { routeItem, canAccessRoute } from "@/lib/menu"
import { roleLabel } from "@/lib/roles"

/** Routes are gated by role; hiding an item in the menu is not enough on its own. */
export function RoleGate({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()
  const { role } = useRole()

  if (canAccessRoute(pathname, role)) {
    return <>{children}</>
  }

  const item = routeItem(pathname)

  return (
    <div className="flex max-w-[520px] items-start gap-3 rounded-lg border border-line-strong bg-card px-4 py-3">
      <Lock size={20} className="mt-0.5 shrink-0 text-primary" aria-hidden />
      <div>
        <h1 className="font-heading text-base">Bu sayfaya erişiminiz yok</h1>
        <p className="mt-1 text-sm leading-relaxed text-foreground">
          {item ? `${item.title} sayfası` : "Bu sayfa"} yalnızca{" "}
          {(item?.roles ?? []).map((allowed) => roleLabel[allowed]).join(", ")}{" "}
          rolüne açıktır. Aktif rolünüz: {roleLabel[role]}.
        </p>
      </div>
    </div>
  )
}
