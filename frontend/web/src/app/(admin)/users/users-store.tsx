"use client"

import * as React from "react"

import { graphqlRequest } from "@/lib/graphql"
import {
  readDemoUsers,
  writeDemoUsers,
  type DemoUser,
} from "@/lib/demo-users"

import {
  departments as departmentSeed,
  departmentIdFrom,
  today,
  users as userSeed,
  type Department,
  type User,
  type UserRole,
  type UserSource,
  type UserStatus,
  type Workstation,
} from "./mock"

/**
 * Kurumun kişi ve departman listesi. Departman taslağı bu arayüzün yerel
 * düzenidir; kimlik, ilk giriş durumu ve cihazlar yetkili yönetim API'sinden
 * düzenli olarak yenilenir. Böylece Desktop'ta tamamlanan ilk doğrulama,
 * sayfa yenilemeden de yönetici ekranına yansır.
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

type OrganizationUsersResponse = {
  organizationUsers: Array<{
    membershipStatus: "INVITED" | "ACTIVE"
    invitedAt: string
    user: {
      id: string
      email: string
      firstName: string
      lastName: string
      status: "ACTIVE" | "INACTIVE" | "SUSPENDED" | "PENDING_DELETION" | "DELETED"
      emailVerified: boolean
    }
    devices: Array<{
      id: string
      name: string
      platform: string
      ipAddress: string
      macAddress: string
      lastSeenAt: string
      createdAt: string
      revokedAt: string | null
    }>
  }>
}

function displayTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return "—"
  return new Intl.DateTimeFormat("tr-TR", {
    dateStyle: "short",
    timeStyle: "short",
  }).format(date)
}

function mapOrganizationUser(
  remote: OrganizationUsersResponse["organizationUsers"][number],
  previous: User | undefined,
  departmentId: string,
): User {
  const workstations: Workstation[] = remote.devices
    .filter((device) => !device.revokedAt)
    .sort((left, right) => right.lastSeenAt.localeCompare(left.lastSeenAt))
    .map((device, index) => ({
      id: device.id,
      hostname: device.name || "Bilinmeyen cihaz",
      ip: device.ipAddress || "—",
      mac: device.macAddress || "—",
      os: device.platform,
      registeredAt: displayTime(device.createdAt),
      lastSeenAt: displayTime(device.lastSeenAt),
      current: index === 0,
    }))
  const latestDevice = remote.devices
    .filter((device) => !device.revokedAt)
    .sort((left, right) => right.lastSeenAt.localeCompare(left.lastSeenAt))[0]
  const active =
    remote.membershipStatus === "ACTIVE" &&
    remote.user.status === "ACTIVE" &&
    remote.user.emailVerified
  const status: UserStatus = active
    ? "active"
    : remote.user.status === "SUSPENDED" || remote.user.status === "DELETED"
      ? "inactive"
      : "invited"
  const name = `${remote.user.firstName} ${remote.user.lastName}`.trim() || remote.user.email

  return {
    id: remote.user.id,
    name,
    email: remote.user.email,
    title: previous?.title ?? "Tanımlanmadı",
    role: previous?.role ?? "employee",
    status,
    departmentId: previous?.departmentId ?? departmentId,
    source: previous?.source ?? "invite",
    addedAt: displayTime(remote.invitedAt),
    lastLoginAt: latestDevice ? displayTime(latestDevice.lastSeenAt) : null,
    workstations,
  }
}

export function UsersProvider({ children }: { children: React.ReactNode }) {
  const [departments, setDepartments] = React.useState<Department[]>(departmentSeed)
  const [users, setUsers] = React.useState<User[]>(userSeed)

  const demoMode = !process.env.NEXT_PUBLIC_GRAPHQL_URL
  const demoUsersHydrated = React.useRef(!demoMode)
  const skipDemoPersist = React.useRef(demoMode)

  React.useEffect(() => {
    if (!demoMode) return

    const timer = window.setTimeout(() => {
      const stored = readDemoUsers()
      if (stored.length > 0) {
        setUsers((previous) => {
          const storedEmails = new Set(stored.map((user) => user.email.toLowerCase()))
          return [
            ...stored,
            ...previous.filter((user) => !storedEmails.has(user.email.toLowerCase())),
          ]
        })
      }
      demoUsersHydrated.current = true
    }, 0)

    return () => window.clearTimeout(timer)
  }, [demoMode])

  React.useEffect(() => {
    if (!demoMode || !demoUsersHydrated.current) return
    if (skipDemoPersist.current) {
      skipDemoPersist.current = false
      return
    }
    writeDemoUsers(users as DemoUser[])
  }, [demoMode, users])

  React.useEffect(() => {
    let disposed = false

    async function refreshOrganizationUsers() {
      if (!window.localStorage.getItem("prova:access-token")) return
      try {
        const data = await graphqlRequest<OrganizationUsersResponse>(
          `query OrganizationUsers {
            organizationUsers {
              membershipStatus
              invitedAt
              user { id email firstName lastName status emailVerified }
              devices {
                id name platform ipAddress macAddress lastSeenAt createdAt revokedAt
              }
            }
          }`,
          {},
        )
        if (disposed) return
        setUsers((previous) => {
          const remoteByEmail = new Map(
            data.organizationUsers.map((record) => [record.user.email.toLowerCase(), record]),
          )
          const retained = previous.filter(
            (user) => !remoteByEmail.has(user.email.toLowerCase()),
          )
          const merged = data.organizationUsers.map((record) =>
            mapOrganizationUser(
              record,
              previous.find(
                (user) => user.email.toLowerCase() === record.user.email.toLowerCase(),
              ),
              departmentSeed[0]?.id ?? "genel",
            ),
          )
          return [...merged, ...retained]
        })
      } catch {
        // Yönetim paneli oturum açılmadan da yerel taslağını gösterebilir.
        // Yetki veya ağ hatası kullanıcı listesini sıfırlamamalıdır.
      }
    }

    void refreshOrganizationUsers()
    const interval = window.setInterval(refreshOrganizationUsers, 15_000)
    window.addEventListener("focus", refreshOrganizationUsers)
    return () => {
      disposed = true
      window.clearInterval(interval)
      window.removeEventListener("focus", refreshOrganizationUsers)
    }
  }, [])

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
