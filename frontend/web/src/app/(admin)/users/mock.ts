import type { Role } from "@/lib/roles"

export type UserStatus = "active" | "invite-pending" | "inactive"

export type User = {
  id: string
  name: string
  email: string
  role: Role | "employee"
  status: UserStatus
  addedAt: string
}

export const userStatusLabel: Record<UserStatus, string> = {
  active: "Aktif",
  "invite-pending": "Davet bekliyor",
  inactive: "Pasif",
}

export const userRoleLabel: Record<User["role"], string> = {
  employee: "Çalışan",
  trainer: "Eğitmen",
  "org-admin": "Kurum yöneticisi",
  platform: "Platform sahibi",
}

export const users: User[] = [
  {
    id: "k-1041",
    name: "Elif Şahin",
    email: "elif.sahin@akbank-egitim.tr",
    role: "org-admin",
    status: "active",
    addedAt: "12.02.2026",
  },
  {
    id: "k-1042",
    name: "Burak Yıldırım",
    email: "burak.yildirim@akbank-egitim.tr",
    role: "trainer",
    status: "active",
    addedAt: "14.02.2026",
  },
  {
    id: "k-1043",
    name: "Ayşe Demirtaş",
    email: "ayse.demirtas@akbank-egitim.tr",
    role: "employee",
    status: "active",
    addedAt: "03.03.2026",
  },
  {
    id: "k-1044",
    name: "Mert Çankaya",
    email: "mert.cankaya@akbank-egitim.tr",
    role: "employee",
    status: "invite-pending",
    addedAt: "28.08.2026",
  },
  {
    id: "k-1045",
    name: "Gizem Aydoğan",
    email: "gizem.aydogan@akbank-egitim.tr",
    role: "employee",
    status: "active",
    addedAt: "11.04.2026",
  },
  {
    id: "k-1046",
    name: "Onur Kılıçarslan",
    email: "onur.kilicarslan@akbank-egitim.tr",
    role: "trainer",
    status: "inactive",
    addedAt: "19.11.2025",
  },
  {
    id: "k-1047",
    name: "Selin Arıkan",
    email: "selin.arikan@akbank-egitim.tr",
    role: "employee",
    status: "active",
    addedAt: "05.06.2026",
  },
]
