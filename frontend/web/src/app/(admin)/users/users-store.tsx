"use client"

import * as React from "react"

import {
  departments as departmentSeed,
  departmentIdFrom,
  today,
  users as userSeed,
  type Department,
  type User,
  type UserRole,
  type UserSource,
} from "./mock"

/**
 * Kurumun kişi ve departman listesi. Arka uç bağlanana kadar tek kaynak
 * burasıdır; sağlayıcı `(admin)` düzeninde durur, böylece kullanıcı ekranları
 * ile atama ekranı aynı listeyi okur: bir kişi silindiğinde ataması da
 * listelerden düşer.
 */
type UsersContextValue = {
  departments: Department[]
  users: User[]
  addUsers: (
    entries: { email: string; name: string; role: UserRole }[],
    departmentId: string,
    source?: UserSource
  ) => number
  toggleStatus: (userId: string) => User | undefined
  revokeWorkstation: (userId: string, workstationId: string) => void
  /** Kişiyi kurumdan tamamen çıkarır; kayıtlı cihazları da birlikte gider. */
  removeUser: (userId: string) => User | undefined
  addDepartment: (input: Omit<Department, "id">) => Department
  /**
   * Departmanı kapatır. Personeli varsa önce `moveTo` departmanına taşınır;
   * kişi silmek ayrı bir karardır, departman kapatmanın yan etkisi değildir.
   */
  removeDepartment: (departmentId: string, moveTo?: string) => void
}

const UsersContext = React.createContext<UsersContextValue | null>(null)

export function UsersProvider({ children }: { children: React.ReactNode }) {
  const [departments, setDepartments] = React.useState<Department[]>(departmentSeed)
  const [users, setUsers] = React.useState<User[]>(userSeed)

  const value = React.useMemo<UsersContextValue>(() => {
    return {
      departments,
      users,

      addUsers(entries, departmentId, source = "bulk") {
        const stamp = today()
        // Aynı adres iki kez eklenmesin; kaynak dosya tekrarlıysa ilki kalır.
        const taken = new Set(users.map((user) => user.email))
        const fresh: User[] = []

        for (const entry of entries) {
          if (taken.has(entry.email)) continue
          taken.add(entry.email)
          fresh.push({
            id: `k-${1100 + users.length + fresh.length + 1}`,
            name: entry.name,
            email: entry.email,
            title: "Tanımlanmadı",
            role: entry.role,
            status: source === "invite" ? "invited" : "expected",
            departmentId,
            source,
            addedAt: stamp,
            lastLoginAt: null,
            workstations: [],
          })
        }

        if (fresh.length > 0) setUsers((previous) => [...fresh, ...previous])
        return fresh.length
      },

      toggleStatus(userId) {
        const target = users.find((user) => user.id === userId)
        setUsers((previous) =>
          previous.map((user) => {
            if (user.id !== userId) return user
            if (user.status === "inactive") {
              // Geri açılan kullanıcı hiç giriş yapmadıysa yeniden bekleyene döner.
              return {
                ...user,
                status: user.lastLoginAt ? "active" : "expected",
              }
            }
            return { ...user, status: "inactive" }
          })
        )
        return target
      },

      revokeWorkstation(userId, workstationId) {
        setUsers((previous) =>
          previous.map((user) =>
            user.id === userId
              ? {
                  ...user,
                  workstations: user.workstations.filter(
                    (workstation) => workstation.id !== workstationId
                  ),
                }
              : user
          )
        )
      },

      removeUser(userId) {
        const target = users.find((user) => user.id === userId)
        if (!target) return undefined
        setUsers((previous) => previous.filter((user) => user.id !== userId))
        return target
      },

      addDepartment(input) {
        const taken = new Set(departments.map((department) => department.id))
        let id = departmentIdFrom(input.name)
        // Aynı adla ikinci bir departman açılabilir; rota parçası çakışmamalı.
        if (taken.has(id)) {
          let suffix = 2
          while (taken.has(`${id}-${suffix}`)) suffix += 1
          id = `${id}-${suffix}`
        }
        const fresh: Department = { id, ...input }
        setDepartments((previous) => [...previous, fresh])
        return fresh
      },

      removeDepartment(departmentId, moveTo) {
        if (moveTo && moveTo !== departmentId) {
          setUsers((previous) =>
            previous.map((user) =>
              user.departmentId === departmentId
                ? { ...user, departmentId: moveTo }
                : user
            )
          )
        }
        setDepartments((previous) =>
          previous.filter((department) => department.id !== departmentId)
        )
      },
    }
  }, [departments, users])

  return <UsersContext.Provider value={value}>{children}</UsersContext.Provider>
}

export function useUsers() {
  const context = React.useContext(UsersContext)
  if (!context) {
    throw new Error("useUsers, UsersProvider içinde kullanılmalı.")
  }
  return context
}
