export type CertificateStatus =
  | "valid"
  | "expiring"
  | "recertification"
  | "expired"

export type Certificate = {
  id: string
  sessionId: string
  employee: string
  scenario: string
  /** The rubric version this certificate was issued against. */
  rubricVersion: string
  issuedAt: string
  validUntil: string
  score: number
  device: string
  status: CertificateStatus
}

export const certificateStatusLabel: Record<CertificateStatus, string> = {
  valid: "Geçerli",
  "expiring": "Süresi doluyor",
  "recertification": "Yeniden sertifikasyon gerekli",
  "expired": "Süresi dolmuş",
}

/** Published rubric versions; a certificate issued against an older one must be renewed. */
export const publishedVersions: Record<string, string> = {
  "Kimliğini vermek istemeyen müşteri": "v3",
  "Öfkeli çağrı merkezi müşterisi": "v2",
  "Denetçi görüşmesi": "v1",
}

export const certificates: Certificate[] = [
  {
    id: "PRV-2026-0148",
    sessionId: "o-5512",
    employee: "Ayşe Demirtaş",
    scenario: "Kimliğini vermek istemeyen müşteri",
    rubricVersion: "v3",
    issuedAt: "03.09.2026",
    validUntil: "03.09.2027",
    score: 92,
    device: "AYSE-MBP-14",
    status: "valid",
  },
  {
    id: "PRV-2026-0147",
    sessionId: "o-5477",
    employee: "Selin Arıkan",
    scenario: "Öfkeli çağrı merkezi müşterisi",
    rubricVersion: "v2",
    issuedAt: "02.09.2026",
    validUntil: "02.09.2027",
    score: 74,
    device: "SUBE-07-ANK",
    status: "valid",
  },
  {
    id: "PRV-2026-0119",
    sessionId: "o-5301",
    employee: "Burak Yıldırım",
    scenario: "Kimliğini vermek istemeyen müşteri",
    rubricVersion: "v2",
    issuedAt: "14.07.2026",
    validUntil: "14.07.2027",
    score: 81,
    device: "BURAK-EGITIM",
    status: "recertification",
  },
  {
    id: "PRV-2026-0112",
    sessionId: "o-5240",
    employee: "Gizem Aydoğan",
    scenario: "Kimliğini vermek istemeyen müşteri",
    rubricVersion: "v2",
    issuedAt: "28.06.2026",
    validUntil: "28.06.2027",
    score: 77,
    device: "SUBE-42-IST",
    status: "recertification",
  },
  {
    id: "PRV-2025-0904",
    sessionId: "o-4912",
    employee: "Onur Kılıçarslan",
    scenario: "Öfkeli çağrı merkezi müşterisi",
    rubricVersion: "v1",
    issuedAt: "19.11.2025",
    validUntil: "19.11.2026",
    score: 88,
    device: "—",
    status: "expiring",
  },
  {
    id: "PRV-2025-0771",
    sessionId: "o-4788",
    employee: "Mert Çankaya",
    scenario: "Öfkeli çağrı merkezi müşterisi",
    rubricVersion: "v1",
    issuedAt: "05.08.2025",
    validUntil: "05.08.2026",
    score: 72,
    device: "—",
    status: "expired",
  },
]

export function needsRenewal() {
  return certificates.filter(
    (certificate) => certificate.status === "recertification"
  )
}
