import type { SheetRead } from "@/lib/xlsx-import"

import type { UserRole } from "./mock"

/**
 * Okunmuş hücre ızgarasını kullanıcı satırlarına çevirir.
 *
 * Ayrıştırma güvenliği `@/lib/xlsx-import` içindedir; burada yapılan iş alan
 * doğrulamasıdır. İki kural burada yaşar:
 *
 *   - `=`, `+`, `-`, `@` ile başlayan hücre veri değil formüldür. Dosya
 *     yeniden dışa aktarıldığında Excel'de çalışacağı için satır elenir.
 *   - Adres RFC'ye tam uymak zorunda değil ama uzunluğu, biçimi ve karakter
 *     kümesi sınırlı olmalı; sınırsız dize hesap kaydına girmez.
 */

const EMAIL_HEADERS = [
  "e-posta",
  "eposta",
  "e-mail",
  "email",
  "mail",
  "e-posta adresi",
  "kurumsal e-posta",
  "adres",
]

const NAME_HEADERS = [
  "ad soyad",
  "adsoyad",
  "ad-soyad",
  "isim",
  "ad ve soyad",
  "personel",
  "name",
  "full name",
]

const FIRST_NAME_HEADERS = ["ad", "isim", "first name", "given name"]
const LAST_NAME_HEADERS = ["soyad", "soyadı", "last name", "surname"]
const ROLE_HEADERS = ["rol", "role", "yetki", "görev tipi"]
const TITLE_HEADERS = ["unvan", "ünvan", "görev", "title", "pozisyon"]

const ROLE_VALUES: Record<string, UserRole> = {
  calisan: "employee",
  personel: "employee",
  employee: "employee",
  yonetici: "org-admin",
  "kurum yoneticisi": "org-admin",
  admin: "org-admin",
  "org-admin": "org-admin",
}

/** 254 adres sınırı; yerel kısım 64. Ardışık nokta ve uç nokta kabul edilmez. */
const EMAIL_PATTERN =
  /^[a-z0-9](?:[a-z0-9._%+-]{0,62}[a-z0-9])?@[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$/

const FORMULA_START = /^[=+\-@\t\r]/

export type CandidateState = "new" | "existing" | "duplicate" | "invalid"

export type Candidate = {
  /** Excel'deki satır numarası; hata mesajı buna bakarak düzeltilir. */
  row: number
  email: string
  name: string
  role: UserRole
  state: CandidateState
  note?: string
}

export type ImportReport = {
  sheetName: string
  candidates: Candidate[]
  /** Kabul edilebilir, yeni satırlar. */
  fresh: Candidate[]
  headerRow: number | null
  truncated: boolean
  columns: { email: number; name: number; first: number; last: number; role: number; title: number }
}

export class MappingError extends Error {
  constructor(message: string) {
    super(message)
    this.name = "MappingError"
  }
}

export function mapSheet(
  sheet: SheetRead,
  existingEmails: Set<string>
): ImportReport {
  const { grid } = sheet
  if (grid.length === 0) {
    throw new MappingError("Sayfada satır yok.")
  }

  const { columns, headerRow } = locateColumns(grid)
  if (columns.email < 0) {
    throw new MappingError(
      "E-posta sütunu bulunamadı. İlk satıra “E-posta” başlığını ekleyin ya da adresleri A sütununa yazın."
    )
  }

  const seen = new Set<string>()
  const candidates: Candidate[] = []
  const start = headerRow === null ? 0 : headerRow + 1

  for (let index = start; index < grid.length; index += 1) {
    const cells = grid[index]
    if (!cells || cells.every((cell) => cell === "")) continue

    const row = index + 1
    const rawEmail = cells[columns.email] ?? ""
    const rawName = readName(cells, columns)

    if (rawEmail === "") continue

    if (FORMULA_START.test(rawEmail) || FORMULA_START.test(rawName)) {
      candidates.push({
        row,
        email: rawEmail.slice(0, 60),
        name: rawName.slice(0, 60),
        role: "employee",
        state: "invalid",
        note: "Formül olabilecek hücre",
      })
      continue
    }

    const email = rawEmail.toLocaleLowerCase("en-US")
    if (
      email.length > 254 ||
      email.includes("..") ||
      !EMAIL_PATTERN.test(email)
    ) {
      candidates.push({
        row,
        email: rawEmail.slice(0, 60),
        name: rawName,
        role: "employee",
        state: "invalid",
        note: "Geçersiz e-posta",
      })
      continue
    }

    const role = readRole(cells, columns)
    const name = cleanName(rawName) || nameFromEmail(email)

    if (seen.has(email)) {
      candidates.push({ row, email, name, role, state: "duplicate", note: "Dosyada tekrar ediyor" })
      continue
    }
    seen.add(email)

    if (existingEmails.has(email)) {
      candidates.push({ row, email, name, role, state: "existing", note: "Zaten kayıtlı" })
      continue
    }

    candidates.push({ row, email, name, role, state: "new" })
  }

  if (candidates.length === 0) {
    throw new MappingError("Sayfada okunabilir bir satır yok.")
  }

  return {
    sheetName: sheet.sheetName,
    candidates,
    fresh: candidates.filter((candidate) => candidate.state === "new"),
    headerRow,
    truncated: sheet.truncated,
    columns,
  }
}

/** Başlık satırı yoksa A sütunu e-posta, B sütunu ad kabul edilir. */
function locateColumns(grid: string[][]) {
  const columns = { email: -1, name: -1, first: -1, last: -1, role: -1, title: -1 }
  const header = grid[0] ?? []

  header.forEach((cell, index) => {
    const key = normalize(cell)
    if (key === "") return
    if (columns.email < 0 && EMAIL_HEADERS.includes(key)) columns.email = index
    else if (columns.name < 0 && NAME_HEADERS.includes(key)) columns.name = index
    else if (columns.first < 0 && FIRST_NAME_HEADERS.includes(key)) columns.first = index
    else if (columns.last < 0 && LAST_NAME_HEADERS.includes(key)) columns.last = index
    else if (columns.role < 0 && ROLE_HEADERS.includes(key)) columns.role = index
    else if (columns.title < 0 && TITLE_HEADERS.includes(key)) columns.title = index
  })

  if (columns.email >= 0) return { columns, headerRow: 0 }

  // Başlıksız dosya: ilk satırda adres varsa doğrudan veri sayılır.
  const firstRow = grid[0] ?? []
  const emailAt = firstRow.findIndex((cell) =>
    EMAIL_PATTERN.test(cell.toLocaleLowerCase("en-US"))
  )
  if (emailAt >= 0) {
    return {
      columns: { ...columns, email: emailAt, name: emailAt === 0 ? 1 : 0 },
      headerRow: null,
    }
  }

  return { columns, headerRow: 0 }
}

function readName(cells: string[], columns: ImportReport["columns"]) {
  if (columns.name >= 0) return cells[columns.name] ?? ""
  if (columns.first >= 0 || columns.last >= 0) {
    return [cells[columns.first] ?? "", cells[columns.last] ?? ""]
      .filter(Boolean)
      .join(" ")
  }
  return ""
}

function readRole(cells: string[], columns: ImportReport["columns"]): UserRole {
  if (columns.role < 0) return "employee"
  // Tanınmayan her değer çalışan sayılır; yükseltme yanlışlıkla verilmez.
  return ROLE_VALUES[normalize(cells[columns.role] ?? "")] ?? "employee"
}

/** Başlık eşlemesi için: küçük harf, aksansız, tek boşluk. */
function normalize(value: string) {
  return value
    .toLocaleLowerCase("tr-TR")
    .replace(/ı/g, "i")
    .normalize("NFD")
    .replace(/\p{Diacritic}/gu, "")
    .replace(/\s+/g, " ")
    .trim()
}

/** Ad alanı yalnızca harf, boşluk, kesme ve tire taşır. */
function cleanName(value: string) {
  return value
    .replace(/[^\p{L}\p{M}\s'’-]/gu, "")
    .replace(/\s+/g, " ")
    .trim()
    .slice(0, 80)
}

function nameFromEmail(email: string) {
  const local = email.split("@")[0] ?? ""
  const parts = local.split(/[._-]+/).filter(Boolean)
  if (parts.length === 0) return email
  return parts
    .map((part) => part.charAt(0).toLocaleUpperCase("tr-TR") + part.slice(1))
    .join(" ")
    .slice(0, 80)
}
