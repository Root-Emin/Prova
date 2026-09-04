/** The three user roles. Employees work in the desktop app, the other two on the web. */
export type Role = "trainer" | "org-admin" | "platform"

export const roleLabel: Record<Role, string> = {
  trainer: "Eğitmen",
  "org-admin": "Kurum yöneticisi",
  platform: "Platform sahibi",
}

export const roleOrder: Role[] = ["trainer", "org-admin", "platform"]
