import { SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar"
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
        defaultOpen
      >
        <AppSidebar />
        <div className="flex min-w-0 flex-1 flex-col overflow-x-auto">
          <header className="flex items-center gap-2 border-b border-sidebar-border px-4 py-3 md:hidden">
            <SidebarTrigger />
            <span className="font-heading text-lg tracking-tight text-primary">
              Prova
            </span>
          </header>
          <div className="prova-icerik prova-kolon px-6 py-6">
            <RoleGate>{children}</RoleGate>
          </div>
        </div>
      </SidebarProvider>
      <Toaster position="bottom-right" />
    </RoleProvider>
  )
}
