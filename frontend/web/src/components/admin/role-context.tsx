"use client"

import * as React from "react"

import type { Role } from "@/lib/roles"

type RoleContextValue = {
  role: Role
  setRole: (role: Role) => void
}

const RoleContext = React.createContext<RoleContextValue | null>(null)

/** Authentication is not built yet; the active role is held here for now. */
export function RoleProvider({
  initialRole = "org-admin",
  children,
}: {
  initialRole?: Role
  children: React.ReactNode
}) {
  const [role, setRole] = React.useState<Role>(initialRole)
  const value = React.useMemo(() => ({ role, setRole }), [role])

  return <RoleContext.Provider value={value}>{children}</RoleContext.Provider>
}

export function useRole() {
  const context = React.useContext(RoleContext)
  if (!context) {
    throw new Error("useRol, RolProvider içinde kullanılmalı.")
  }
  return context
}
