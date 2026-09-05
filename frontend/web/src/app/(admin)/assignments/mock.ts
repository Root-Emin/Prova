import { scenarios } from "../personas/mock"
import type { PersonaColor } from "../personas/persona-colors"

export type AssignmentStatus = "pending" | "completed" | "overdue"

export type Assignment = {
  id: string
  /** Kişi kaydı tek kaynaktır; atama adı değil kullanıcı kimliğini taşır. */
  userId: string
  scenarioId: string
  scenario: string
  version: string
  assignedBy: string
  assignedAt: string
  dueDate: string
  status: AssignmentStatus
  /** Yeniden sertifikasyon kapsamında açılan atamalar işaretlenir. */
  renewal?: boolean
}

export const assignmentStatusLabel: Record<AssignmentStatus, string> = {
  pending: "Bekliyor",
  completed: "Tamamlandı",
  overdue: "Gecikti",
}

export type AssignableScenario = {
  id: string
  /** Senaryonun kişilik rengi; atama diyaloğu önce bu renge göre süzer. */
  color: PersonaColor
  name: string
  persona: string
  sector: string
  /** Atama anında yayında olan sürüm; oturum bu sürümle oynanır. */
  version: string
}

/**
 * Atanabilir senaryolar personalar sekmesinden gelir: yalnızca yayındaki
 * senaryolar sınav olur, taslak sürüm atanamaz.
 */
export const assignableScenarios: AssignableScenario[] = scenarios
  .filter((scenario) => scenario.status === "published")
  .map((scenario) => ({
    id: scenario.id,
    color: scenario.color,
    name: scenario.name,
    persona: scenario.personaName,
    sector: scenario.sector,
    version:
      scenario.versions.find((version) => version.published)?.label ??
      scenario.versions[0].label,
  }))

export const assignments: Assignment[] = [
  {
    id: "a-311",
    userId: "k-1044",
    scenarioId: "s-202",
    scenario: "Öfkeli çağrı merkezi müşterisi",
    version: "v2",
    assignedBy: "Burak Yıldırım",
    assignedAt: "28.08.2026",
    dueDate: "05.09.2026",
    status: "pending",
  },
  {
    id: "a-310",
    userId: "k-1045",
    scenarioId: "s-201",
    scenario: "Kimliğini vermek istemeyen müşteri",
    version: "v3",
    assignedBy: "Burak Yıldırım",
    assignedAt: "27.08.2026",
    dueDate: "03.09.2026",
    status: "completed",
  },
  {
    id: "a-309",
    userId: "k-1047",
    scenarioId: "s-203",
    scenario: "Denetçi görüşmesi",
    version: "v1",
    assignedBy: "Elif Şahin",
    assignedAt: "21.08.2026",
    dueDate: "31.08.2026",
    status: "overdue",
  },
  {
    id: "a-307",
    userId: "k-1046",
    scenarioId: "s-202",
    scenario: "Öfkeli çağrı merkezi müşterisi",
    version: "v2",
    assignedBy: "Elif Şahin",
    assignedAt: "18.08.2026",
    dueDate: "10.09.2026",
    status: "pending",
    renewal: true,
  },
]

/** Tamamlanmamış atama açık sayılır; kişi yeni sınav listesine düşmez. */
export function isOpen(assignment: Assignment) {
  return assignment.status !== "completed"
}

/** "05.09.2026" → Date. Mock veriler tek biçimde tutulur. */
function parseDate(value: string) {
  const [day, month, year] = value.split(".").map(Number)
  return new Date(year, (month ?? 1) - 1, day ?? 1)
}

export function formatDate(value: Date) {
  return value.toLocaleDateString("tr-TR", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  })
}

/**
 * Son tarihe kalan süre. Tarihin kendisi tek başına "yetişir mi" sorusunu
 * cevaplamıyor; satırda okunan asıl bilgi bu.
 */
export function dueHint(assignment: Assignment, now = new Date()) {
  if (assignment.status === "completed") return null
  const due = parseDate(assignment.dueDate)
  const day = 24 * 60 * 60 * 1000
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const days = Math.round((due.getTime() - today.getTime()) / day)

  if (days < 0) return `${Math.abs(days)} gün gecikti`
  if (days === 0) return "bugün son gün"
  if (days === 1) return "yarın son gün"
  return `${days} gün kaldı`
}
