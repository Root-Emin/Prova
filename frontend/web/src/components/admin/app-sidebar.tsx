"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarSeparator,
} from "@/components/ui/sidebar"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { useRole } from "@/components/admin/role-context"
import { isActive, groupLabel, menu, type MenuGroup } from "@/lib/menu"
import { roleLabel, roleOrder, type Role } from "@/lib/roles"

const roleOptions = Object.fromEntries(
  roleOrder.map((role) => [role, roleLabel[role]])
)

export function AppSidebar() {
  const pathname = usePathname()
  const { role, setRole } = useRole()

  return (
    <Sidebar collapsible="offcanvas" className="border-r border-sidebar-border">
      <SidebarHeader className="gap-0 px-3 pt-4 pb-3">
        <span className="font-heading text-xl tracking-tight text-primary">
          Prova
        </span>
        <span className="prova-meta uppercase">Anadolu Katılım Bankası</span>
      </SidebarHeader>

      <SidebarSeparator className="mx-0" />

      <SidebarContent>
        {(["training", "organization"] as MenuGroup[]).map((group) => {
          const items = menu.filter(
            (item) => item.group === group && item.roles.includes(role)
          )
          if (items.length === 0) return null

          return (
            <SidebarGroup key={group}>
              <SidebarGroupLabel>{groupLabel[group]}</SidebarGroupLabel>
              <SidebarGroupContent>
                <SidebarMenu>
                  {items.map((item) => (
                    <SidebarMenuItem key={item.path}>
                      <SidebarMenuButton
                        isActive={isActive(pathname, item.path)}
                        render={<Link href={item.path} />}
                      >
                        <item.icon aria-hidden />
                        <span>{item.title}</span>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  ))}
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>
          )
        })}
      </SidebarContent>

      <SidebarFooter className="gap-2 border-t border-sidebar-border px-3 py-3">
        <span className="prova-meta uppercase">Aktif rol</span>
        <Select
          items={roleOptions}
          value={role}
          onValueChange={(value) => setRole(value as Role)}
        >
          <SelectTrigger className="w-full bg-card">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {roleOrder.map((option) => (
              <SelectItem key={option} value={option}>
                {roleLabel[option]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <p className="prova-meta normal-case">
          Rotalar role göre kapılıdır; menü buna göre değişir.
        </p>
      </SidebarFooter>
    </Sidebar>
  )
}
