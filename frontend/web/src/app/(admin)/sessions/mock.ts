import type { RubricStatus } from "@/lib/rubric"

export type SessionStatus = "evaluated" | "evaluating"

export type SessionSummary = {
  id: string
  employee: string
  scenario: string
  version: string
  date: string
  duration: string
  device: string
  status: SessionStatus
  /** Filled in once evaluation has finished. */
  score?: number
  result?: RubricStatus
  mandatoryFailed?: boolean
  overridden?: boolean
}

export const sessionSummaries: SessionSummary[] = [
  {
    id: "o-5513",
    employee: "Gizem Aydoğan",
    scenario: "Kimliğini vermek istemeyen müşteri",
    version: "v3",
    date: "03.09.2026 09:44",
    duration: "07:12",
    device: "SUBE-42-IST",
    status: "evaluating",
  },
  {
    id: "o-5512",
    employee: "Ayşe Demirtaş",
    scenario: "Kimliğini vermek istemeyen müşteri",
    version: "v3",
    date: "03.09.2026 09:02",
    duration: "05:48",
    device: "AYSE-MBP-14",
    status: "evaluated",
    score: 92,
    result: "passed",
    mandatoryFailed: false,
    overridden: true,
  },
  {
    id: "o-5498",
    employee: "Gizem Aydoğan",
    scenario: "Kimliğini vermek istemeyen müşteri",
    version: "v3",
    date: "02.09.2026 16:20",
    duration: "06:41",
    device: "SUBE-42-IST",
    status: "evaluated",
    score: 58,
    result: "failed",
    mandatoryFailed: true,
    overridden: false,
  },
  {
    id: "o-5477",
    employee: "Selin Arıkan",
    scenario: "Öfkeli çağrı merkezi müşterisi",
    version: "v2",
    date: "02.09.2026 11:02",
    duration: "08:03",
    device: "SUBE-07-ANK",
    status: "evaluated",
    score: 74,
    result: "passed",
    mandatoryFailed: false,
    overridden: false,
  },
  {
    id: "o-5460",
    employee: "Mert Çankaya",
    scenario: "Öfkeli çağrı merkezi müşterisi",
    version: "v2",
    date: "01.09.2026 14:35",
    duration: "04:19",
    device: "SUBE-07-ANK",
    status: "evaluated",
    score: 41,
    result: "failed",
    mandatoryFailed: true,
    overridden: false,
  },
  {
    id: "o-5431",
    employee: "Ayşe Demirtaş",
    scenario: "Denetçi görüşmesi",
    version: "v1",
    date: "31.08.2026 10:11",
    duration: "09:27",
    device: "AYSE-MBP-14",
    status: "evaluating",
  },
]
