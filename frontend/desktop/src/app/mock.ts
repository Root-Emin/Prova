import type { RubricStatus } from "@/lib/rubric"

export const sessionBriefing = {
  scenario: "Kimliğini vermek istemeyen müşteri",
  persona: "Necdet Bey, 58",
  version: "v3",
  assignedBy: "Burak Yıldırım · eğitmen",
  estimatedDuration: "6–8 dakika",
  passThreshold: 70,
}

export type AssignedSession = {
  id: string
  scenario: string
  persona: string
  version: string
  dueDate: string
  assignedBy: string
  renewal?: boolean
}

export type PastSession = {
  id: string
  scenario: string
  version: string
  date: string
  score: number
  result: RubricStatus
  certificateNo?: string
}

export const assignedSessions: AssignedSession[] = [
  {
    id: "a-310",
    scenario: "Kimliğini vermek istemeyen müşteri",
    persona: "Necdet Bey, 58",
    version: "v3",
    dueDate: "05.09.2026",
    assignedBy: "Burak Yıldırım",
  },
  {
    id: "a-307",
    scenario: "Öfkeli çağrı merkezi müşterisi",
    persona: "Deniz Hanım, 34",
    version: "v2",
    dueDate: "10.09.2026",
    assignedBy: "Elif Şahin",
    renewal: true,
  },
]

export const pastSessions: PastSession[] = [
  {
    id: "o-5498",
    scenario: "Kimliğini vermek istemeyen müşteri",
    version: "v3",
    date: "02.09.2026",
    score: 58,
    result: "failed",
  },
  {
    id: "o-5390",
    scenario: "Öfkeli çağrı merkezi müşterisi",
    version: "v2",
    date: "12.08.2026",
    score: 81,
    result: "passed",
    certificateNo: "PRV-2026-0131",
  },
  {
    id: "o-5288",
    scenario: "Denetçi görüşmesi",
    version: "v1",
    date: "24.07.2026",
    score: 76,
    result: "passed",
    certificateNo: "PRV-2026-0108",
  },
]
