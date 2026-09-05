import {
  Award,
  Bot,
  ClipboardList,
  Laptop,
  LayoutDashboard,
  MessagesSquare,
  ScrollText,
  Users,
  type LucideIcon,
} from "lucide-react"

export type MenuGroup = "training" | "organization"

export type MenuLink = {
  title: string
  path: string
  icon: LucideIcon
  group: MenuGroup
}

export const groupLabel: Record<MenuGroup, string> = {
  training: "Eğitim",
  organization: "Kurum",
}

/** Kenar çubuğu bu listeden kurulur. Paneli tek rol kullanır: kurum yöneticisi. */
export const menu: MenuLink[] = [
  { title: "Panel", path: "/", icon: LayoutDashboard, group: "training" },
  { title: "Personalar", path: "/personas", icon: Bot, group: "training" },
  { title: "Atamalar", path: "/assignments", icon: ClipboardList, group: "training" },
  { title: "Oturumlar", path: "/sessions", icon: MessagesSquare, group: "training" },
  { title: "Sertifikalar", path: "/certificates", icon: Award, group: "training" },
  { title: "Kullanıcılar", path: "/users", icon: Users, group: "organization" },
  { title: "Cihazlar", path: "/devices", icon: Laptop, group: "organization" },
  { title: "Denetim kaydı", path: "/audit-log", icon: ScrollText, group: "organization" },
]

export function isActive(pathname: string, path: string) {
  return path === "/" ? pathname === "/" : pathname === path || pathname.startsWith(`${path}/`)
}
