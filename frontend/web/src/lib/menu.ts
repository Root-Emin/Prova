import {
  Award,
  Bot,
  ClipboardList,
  Laptop,
  LayoutDashboard,
  MessagesSquare,
  ScrollText,
  Split,
  Users,
  type LucideIcon,
} from "lucide-react"

import type { Role } from "@/lib/roles"

export type MenuGroup = "training" | "organization"

export type MenuLink = {
  title: string
  path: string
  icon: LucideIcon
  roles: Role[]
  group: MenuGroup
}

export const groupLabel: Record<MenuGroup, string> = {
  training: "Eğitim",
  organization: "Kurum",
}

const allRoles: Role[] = ["trainer", "org-admin", "platform"]

/** The menu and the route gate are fed by this one list. */
export const menu: MenuLink[] = [
  { title: "Panel", path: "/", icon: LayoutDashboard, roles: allRoles, group: "training" },
  { title: "Personalar", path: "/personas", icon: Bot, roles: allRoles, group: "training" },
  { title: "Atamalar", path: "/assignments", icon: ClipboardList, roles: allRoles, group: "training" },
  { title: "Oturumlar", path: "/sessions", icon: MessagesSquare, roles: allRoles, group: "training" },
  { title: "Sertifikalar", path: "/certificates", icon: Award, roles: allRoles, group: "training" },
  {
    title: "Kullanıcılar",
    path: "/users",
    icon: Users,
    roles: ["org-admin", "platform"],
    group: "organization",
  },
  {
    title: "Cihazlar",
    path: "/devices",
    icon: Laptop,
    roles: ["org-admin", "platform"],
    group: "organization",
  },
  {
    title: "LLM dağılımı",
    path: "/llm-routing",
    icon: Split,
    roles: ["platform"],
    group: "organization",
  },
  {
    title: "Denetim kaydı",
    path: "/audit-log",
    icon: ScrollText,
    roles: ["org-admin", "platform"],
    group: "organization",
  },
]

function matchRoute(path: string) {
  // The longest matching prefix wins; "/" matches only itself.
  return [...menu]
    .sort((a, b) => b.path.length - a.path.length)
    .find((item) =>
      item.path === "/" ? path === "/" : path === item.path || path.startsWith(`${item.path}/`)
    )
}

export function canAccessRoute(path: string, role: Role) {
  const item = matchRoute(path)
  return item ? item.roles.includes(role) : true
}

export function routeItem(path: string) {
  return matchRoute(path)
}

export function isActive(pathname: string, path: string) {
  return path === "/" ? pathname === "/" : pathname === path || pathname.startsWith(`${path}/`)
}
