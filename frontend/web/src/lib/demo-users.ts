export type DemoUser = {
  id: string
  name: string
  email: string
  title: string
  role: "employee" | "org-admin"
  status: "active" | "inactive" | "invited" | "expected"
  departmentId: string
  source: "invite" | "bulk"
  addedAt: string
  lastLoginAt: string | null
  workstations: []
}

const storageKey = "prova:demo-users"

export function readDemoUsers(): DemoUser[] {
  if (typeof window === "undefined") return []
  try {
    const value = JSON.parse(window.localStorage.getItem(storageKey) ?? "[]")
    return Array.isArray(value) ? (value as DemoUser[]) : []
  } catch {
    return []
  }
}

export function writeDemoUsers(users: DemoUser[]): void {
  if (typeof window === "undefined") return
  window.localStorage.setItem(storageKey, JSON.stringify(users))
}

