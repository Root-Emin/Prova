export type AssignmentStatus = "pending" | "completed" | "overdue"

export type Assignment = {
  id: string
  employee: string
  scenario: string
  version: string
  assignedBy: string
  assignedAt: string
  dueDate: string
  status: AssignmentStatus
  /** Set when the assignment was opened as part of a recertification. */
  renewal?: boolean
}

export const assignmentStatusLabel: Record<AssignmentStatus, string> = {
  pending: "Bekliyor",
  completed: "Tamamlandı",
  overdue: "Gecikti",
}

export const assignments: Assignment[] = [
  {
    id: "a-311",
    employee: "Mert Çankaya",
    scenario: "Öfkeli çağrı merkezi müşterisi",
    version: "v2",
    assignedBy: "Burak Yıldırım",
    assignedAt: "28.08.2026",
    dueDate: "05.09.2026",
    status: "pending",
  },
  {
    id: "a-310",
    employee: "Gizem Aydoğan",
    scenario: "Kimliğini vermek istemeyen müşteri",
    version: "v3",
    assignedBy: "Burak Yıldırım",
    assignedAt: "27.08.2026",
    dueDate: "03.09.2026",
    status: "completed",
  },
  {
    id: "a-309",
    employee: "Selin Arıkan",
    scenario: "Denetçi görüşmesi",
    version: "v1",
    assignedBy: "Elif Şahin",
    assignedAt: "21.08.2026",
    dueDate: "31.08.2026",
    status: "overdue",
  },
  {
    id: "a-308",
    employee: "Ayşe Demirtaş",
    scenario: "Kimliğini vermek istemeyen müşteri",
    version: "v3",
    assignedBy: "Burak Yıldırım",
    assignedAt: "20.08.2026",
    dueDate: "03.09.2026",
    status: "completed",
  },
  {
    id: "a-307",
    employee: "Onur Kılıçarslan",
    scenario: "Öfkeli çağrı merkezi müşterisi",
    version: "v2",
    assignedBy: "Elif Şahin",
    assignedAt: "18.08.2026",
    dueDate: "10.09.2026",
    status: "pending",
    renewal: true,
  },
]

export const assignableEmployees = [
  "Ayşe Demirtaş",
  "Gizem Aydoğan",
  "Selin Arıkan",
  "Mert Çankaya",
  "Onur Kılıçarslan",
]

export const assignableScenarios = [
  { name: "Kimliğini vermek istemeyen müşteri", version: "v3" },
  { name: "Öfkeli çağrı merkezi müşterisi", version: "v2" },
  { name: "Denetçi görüşmesi", version: "v1" },
]
