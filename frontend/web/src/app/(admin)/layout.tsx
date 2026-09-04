import { SidebarProvider } from "@/components/ui/sidebar"
import { Toaster } from "@/components/ui/sonner"
import { AppSidebar } from "@/components/admin/app-sidebar"
import { RoleGate } from "@/components/admin/role-gate"
import { RoleProvider } from "@/components/admin/role-context"

/** Admin shell: narrow left sidebar, cream ground, no texture. */
export default function AdminLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <RoleProvider>
      <SidebarProvider
        style={{ "--sidebar-width": "220px" } as React.CSSProperties}
        className="min-h-svh"
      >
        <AppSidebar />
        <div className="flex-1 overflow-x-hidden">
          <div className="prova-icerik prova-kolon px-6 py-6">
            <RoleGate>{children}</RoleGate>
          </div>
        </div>
      </SidebarProvider>
      <Toaster position="bottom-right" />
    </RoleProvider>
  )
}
