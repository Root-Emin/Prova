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
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { ProvaLogo } from "@/components/prova/logo"
import { currentAdmin } from "@/lib/current-admin"
import { isActive, groupLabel, menu, type MenuGroup } from "@/lib/menu"

import { initialsOf } from "@/app/(admin)/users/mock"

export function AppSidebar() {
  const pathname = usePathname()

  return (
    <Sidebar collapsible="offcanvas" className="border-r border-sidebar-border">
      <SidebarHeader className="gap-1.5 px-3 pt-4 pb-3">
        <ProvaLogo size="md" />
        {/* The tenant name gets its own line; beside the mark it truncates. */}
        <span className="prova-meta uppercase">Anadolu Katılım Bankası</span>
      </SidebarHeader>

      <SidebarSeparator className="mx-0" />

      <SidebarContent>
        {(["training", "organization"] as MenuGroup[]).map((group) => {
          const items = menu.filter((item) => item.group === group)

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
        <span className="prova-meta uppercase">Oturum</span>
        <div className="flex min-w-0 items-center gap-2.5">
          <Avatar>
            <AvatarFallback>{initialsOf(currentAdmin.name)}</AvatarFallback>
          </Avatar>
          <div className="min-w-0">
            <div className="truncate text-sm font-medium">
              {currentAdmin.name}
            </div>
            <div className="prova-meta truncate normal-case">
              Kurum yöneticisi
            </div>
          </div>
        </div>
      </SidebarFooter>
    </Sidebar>
  )
}
