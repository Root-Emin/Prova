/**
 * Tarayıcı içinde .xlsx okuyucu.
 *
 * Dosya sunucuya gitmez; arşivi açmak ve hücreleri okumak burada olur. Dış
 * bağımlılık yoktur: .xlsx bir ZIP olduğu için arşiv tarayıcının kendi
 * `DecompressionStream`'iyle açılır, sayfa `DOMParser` ile okunur. Böylece
 * yükleme yüzeyine üçüncü taraf bir ayrıştırıcı girmez.
 *
 * Kabul kuralları sırayla uygulanır, biri düşerse dosya hiç açılmaz:
 *
 *  1. Uzantı tam olarak `.xlsx` — `.xls`, `.xlsm`, `.csv` ayrı mesajla reddedilir
 *  2. Tarayıcının verdiği MIME tipi xlsx ya da boş
 *  3. Boyut 0 < n <= 2 MB
 *  4. İlk dört bayt `PK\x03\x04` — uzantısı değiştirilmiş dosya burada elenir
 *  5. Girdi sayısı, girdi başına ve toplam açılmış boyut sınırlı (zip bomb)
 *  6. Girdi adında `..`, mutlak yol veya ters bölü yok (path traversal)
 *  7. Sıkıştırma yöntemi yalnızca store (0) veya deflate (8)
 *  8. `[Content_Types].xml` ve `xl/workbook.xml` var — adı değiştirilmiş .docx elenir
 *  9. Makro (`vbaProject.bin`) ve dış bağlantı (`externalLinks/`) içeren dosya reddedilir
 * 10. Hücre metni denetim karakterlerinden arındırılır ve uzunluğu kırpılır
 *
 * Formül enjeksiyonu (`=`, `+`, `-`, `@` ile başlayan hücre) burada değil alan
 * eşlemesinde satır bazında elenir; hangi satırın neden düştüğü kullanıcıya
 * gösterilebilsin diye.
 */

const XLSX_MIME =
  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

const LIMITS = {
  /** Kurumsal bir personel listesi bunun yanına yaklaşmaz. */
  fileBytes: 2 * 1024 * 1024,
  entries: 64,
  entryBytes: 8 * 1024 * 1024,
  totalBytes: 24 * 1024 * 1024,
  rows: 5000,
  columns: 32,
  cellChars: 320,
} as const

export class XlsxError extends Error {
  constructor(message: string) {
    super(message)
    this.name = "XlsxError"
  }
}

export type SheetRead = {
  sheetName: string
  /** Satır satır hücre metinleri. Boş hücreler boş dizedir. */
  grid: string[][]
  /** Satır sınırı aşıldıysa okuma kesilmiştir. */
  truncated: boolean
}

/** Dosya seçicinin `accept` değeri; tek yerden gelsin. */
export const XLSX_ACCEPT = `.xlsx,${XLSX_MIME}`

export const XLSX_MAX_BYTES = LIMITS.fileBytes

const EXTENSION_HINTS: Record<string, string> = {
  ".xls":
    "Eski .xls biçimi kabul edilmiyor. Excel'de “Farklı kaydet → .xlsx” ile kaydedin.",
  ".xlsm": "Makro içeren .xlsm kabul edilmiyor. Makrosuz .xlsx olarak kaydedin.",
  ".xlsb": "İkili .xlsb biçimi kabul edilmiyor. .xlsx olarak kaydedin.",
  ".csv": "CSV kabul edilmiyor. Dosyayı .xlsx olarak kaydedin.",
  ".numbers": "Numbers dosyası kabul edilmiyor. .xlsx olarak dışa aktarın.",
  ".ods": "OpenDocument kabul edilmiyor. .xlsx olarak kaydedin.",
}

/**
 * Dosya açılmadan önceki ucuz kontroller: ad, tip, boyut. Baytlara bakmaz.
 */
export function assertAcceptableFile(file: File) {
  const name = file.name.split(/[\\/]/).pop() ?? ""

  if (name.length === 0 || name.length > 200) {
    throw new XlsxError("Dosya adı geçersiz.")
  }

  if (/[\u0000-\u001f\u007f]/.test(name)) {
    throw new XlsxError("Dosya adı geçersiz karakter içeriyor.")
  }

  const dot = name.lastIndexOf(".")
  const extension = dot === -1 ? "" : name.slice(dot).toLowerCase()
  if (extension !== ".xlsx") {
    throw new XlsxError(
      EXTENSION_HINTS[extension] ??
        "Yalnızca .xlsx uzantılı Excel dosyası yüklenebilir."
    )
  }

  // Tarayıcı tipi boş bırakabilir; doluysa xlsx olmak zorunda.
  if (file.type && file.type !== XLSX_MIME) {
    throw new XlsxError("Dosya türü Excel çalışma kitabı değil.")
  }

  if (file.size === 0) {
    throw new XlsxError("Dosya boş.")
  }
  if (file.size > LIMITS.fileBytes) {
    throw new XlsxError(
      `Dosya ${(file.size / 1024 / 1024).toFixed(1)} MB. Sınır ${
        LIMITS.fileBytes / 1024 / 1024
      } MB.`
    )
  }
}

/** Dosyanın ilk sayfasını hücre ızgarası olarak okur. */
export async function readFirstSheet(file: File): Promise<SheetRead> {
  assertAcceptableFile(file)

  if (typeof DecompressionStream === "undefined") {
    throw new XlsxError(
      "Bu tarayıcı arşiv açmayı desteklemiyor. Güncel bir sürüm gerekiyor."
    )
  }

  const bytes = new Uint8Array(await file.arrayBuffer())
  if (bytes.byteLength < 22) {
    throw new XlsxError("Dosya bozuk.")
  }
  // 4. kural: uzantı değil, içerik karar verir.
  if (
    bytes[0] !== 0x50 ||
    bytes[1] !== 0x4b ||
    bytes[2] !== 0x03 ||
    bytes[3] !== 0x04
  ) {
    throw new XlsxError(
      "Dosyanın içeriği Excel çalışma kitabı değil; uzantısı değiştirilmiş olabilir."
    )
  }

  const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength)
  const entries = readCentralDirectory(view, bytes)
  assertNoDangerousParts(entries)

  const archive = { bytes, view, entries }
  const workbook = await readXml(archive, "xl/workbook.xml", "Çalışma kitabı")
  const relations = entries.has("xl/_rels/workbook.xml.rels")
    ? await readXml(archive, "xl/_rels/workbook.xml.rels", "İlişki tablosu")
    : null

  const sheetEntry = entries.get(
    resolveFirstSheetPath(entries, workbook, relations)
  )
  if (!sheetEntry) {
    throw new XlsxError("Çalışma kitabında okunabilir bir sayfa yok.")
  }

  const shared = entries.has("xl/sharedStrings.xml")
    ? readSharedStrings(
        await readXml(archive, "xl/sharedStrings.xml", "Metin tablosu")
      )
    : []

  const sheet = parseXml(await readEntry(archive, sheetEntry), "Sayfa")

  return { sheetName: sheetNameOf(workbook), ...readGrid(sheet, shared) }
}

/* ------------------------------------------------------------------ ZIP */

type ZipEntry = {
  name: string
  method: number
  compressedSize: number
  size: number
  localOffset: number
}

type Archive = {
  bytes: Uint8Array
  view: DataView
  entries: Map<string, ZipEntry>
}

function readCentralDirectory(view: DataView, bytes: Uint8Array) {
  const eocd = findEndOfCentralDirectory(view)
  const count = view.getUint16(eocd + 10, true)
  const dirSize = view.getUint32(eocd + 12, true)
  const dirOffset = view.getUint32(eocd + 16, true)

  if (count === 0) throw new XlsxError("Arşiv boş.")
  if (count > LIMITS.entries) {
    throw new XlsxError("Arşivde beklenenden çok dosya var.")
  }
  if (dirOffset + dirSize > view.byteLength) {
    throw new XlsxError("Arşiv bozuk.")
  }

  const decoder = new TextDecoder("utf-8")
  const entries = new Map<string, ZipEntry>()
  let cursor = dirOffset
  let declaredTotal = 0

  for (let index = 0; index < count; index += 1) {
    if (cursor + 46 > view.byteLength) throw new XlsxError("Arşiv bozuk.")
    if (view.getUint32(cursor, true) !== 0x02014b50) {
      throw new XlsxError("Arşiv bozuk.")
    }

    const method = view.getUint16(cursor + 10, true)
    const compressedSize = view.getUint32(cursor + 20, true)
    const size = view.getUint32(cursor + 24, true)
    const nameLength = view.getUint16(cursor + 28, true)
    const extraLength = view.getUint16(cursor + 30, true)
    const commentLength = view.getUint16(cursor + 32, true)
    const localOffset = view.getUint32(cursor + 42, true)
    const name = decoder.decode(
      bytes.subarray(cursor + 46, cursor + 46 + nameLength)
    )

    assertSafeEntryName(name)
    if (method !== 0 && method !== 8) {
      throw new XlsxError("Arşivde desteklenmeyen sıkıştırma var.")
    }
    if (size > LIMITS.entryBytes) {
      throw new XlsxError("Arşivdeki bir dosya sınırın üzerinde.")
    }
    declaredTotal += size
    if (declaredTotal > LIMITS.totalBytes) {
      throw new XlsxError("Arşiv açıldığında sınırın üzerinde büyüyor.")
    }
    if (localOffset + 30 > view.byteLength) throw new XlsxError("Arşiv bozuk.")

    entries.set(name, { name, method, compressedSize, size, localOffset })
    cursor += 46 + nameLength + extraLength + commentLength
  }

  return entries
}

function findEndOfCentralDirectory(view: DataView) {
  const earliest = Math.max(0, view.byteLength - 22 - 0xffff)
  for (let offset = view.byteLength - 22; offset >= earliest; offset -= 1) {
    if (view.getUint32(offset, true) === 0x06054b50) return offset
  }
  throw new XlsxError("Dosya geçerli bir Excel arşivi değil.")
}

function assertSafeEntryName(name: string) {
  if (
    name.length === 0 ||
    name.length > 200 ||
    name.startsWith("/") ||
    name.includes("\\") ||

    /[\u0000-\u001f]/.test(name) ||
    name.split("/").includes("..")
  ) {
    throw new XlsxError("Arşivde güvenli olmayan bir dosya yolu var.")
  }
}

/** Makro ve dış veri bağlantısı taşıyan çalışma kitabı hiç açılmaz. */
function assertNoDangerousParts(entries: Map<string, ZipEntry>) {
  for (const name of entries.keys()) {
    if (name.endsWith("vbaProject.bin")) {
      throw new XlsxError("Dosya makro içeriyor. Makrosuz .xlsx yükleyin.")
    }
    if (name.startsWith("xl/externalLinks/")) {
      throw new XlsxError(
        "Dosya dış veri bağlantısı içeriyor. Bağlantıları kaldırıp yeniden kaydedin."
      )
    }
    if (name.startsWith("xl/embeddings/") || name.startsWith("xl/activeX/")) {
      throw new XlsxError("Dosya gömülü nesne içeriyor. Nesneleri kaldırın.")
    }
  }
  if (!entries.has("[Content_Types].xml") || !entries.has("xl/workbook.xml")) {
    throw new XlsxError("Dosya bir Excel çalışma kitabı değil.")
  }
}

async function readEntry(
  { bytes, view }: Archive,
  entry: ZipEntry
): Promise<Uint8Array> {
  const header = entry.localOffset
  if (view.getUint32(header, true) !== 0x04034b50) {
    throw new XlsxError("Arşiv bozuk.")
  }
  // Yerel başlıktaki ad/ek alan uzunlukları merkezî dizindekinden farklı olabilir.
  const nameLength = view.getUint16(header + 26, true)
  const extraLength = view.getUint16(header + 28, true)
  const start = header + 30 + nameLength + extraLength
  const end = start + entry.compressedSize
  if (end > bytes.byteLength) throw new XlsxError("Arşiv bozuk.")

  const raw = bytes.subarray(start, end)
  if (entry.method === 0) return raw

  const stream = new Blob([raw as BlobPart])
    .stream()
    .pipeThrough(new DecompressionStream("deflate-raw"))
  const reader = stream.getReader()
  const chunks: Uint8Array[] = []
  let total = 0

  for (;;) {
    const { done, value } = await reader.read()
    if (done) break
    total += value.byteLength
    // Bildirilen boyuta değil, gerçekten akana bakılır.
    if (total > LIMITS.entryBytes) {
      await reader.cancel()
      throw new XlsxError("Arşiv açıldığında sınırın üzerinde büyüyor.")
    }
    chunks.push(value)
  }

  const out = new Uint8Array(total)
  let cursor = 0
  for (const chunk of chunks) {
    out.set(chunk, cursor)
    cursor += chunk.byteLength
  }
  return out
}

/* ------------------------------------------------------------------ XML */

const RELS_NS =
  "http://schemas.openxmlformats.org/officeDocument/2006/relationships"

async function readXml(archive: Archive, path: string, label: string) {
  const entry = archive.entries.get(path)
  if (!entry) throw new XlsxError(`${label} bulunamadı.`)
  return parseXml(await readEntry(archive, entry), label)
}

function parseXml(bytes: Uint8Array, label: string): Document {
  const text = new TextDecoder("utf-8").decode(bytes)
  // application/xml: betik çalışmaz, dış varlık çözülmez.
  const doc = new DOMParser().parseFromString(text, "application/xml")
  if (doc.querySelector("parsererror")) {
    throw new XlsxError(`${label} okunamadı; dosya bozuk olabilir.`)
  }
  return doc
}

function sheetNameOf(workbook: Document) {
  return (
    workbook.querySelector("sheets > sheet")?.getAttribute("name") ?? "Sayfa1"
  )
}

/**
 * İlk sayfanın arşiv içindeki yolu. Sayfa sırası `workbook.xml`'de, dosya adı
 * ilişki tablosundadır; ikisi eşleşmezse yanlış sayfa okunur.
 */
function resolveFirstSheetPath(
  entries: Map<string, ZipEntry>,
  workbook: Document,
  relations: Document | null
) {
  const sheet = workbook.querySelector("sheets > sheet")
  const relationId =
    sheet?.getAttributeNS(RELS_NS, "id") ?? sheet?.getAttribute("r:id") ?? null

  if (relationId && relations) {
    for (const relation of Array.from(
      relations.querySelectorAll("Relationship")
    )) {
      if (relation.getAttribute("Id") !== relationId) continue
      const target = (relation.getAttribute("Target") ?? "").replace(/^\/+/, "")
      // Hedef `xl/` klasörüne görelidir; mutlak biçim de kullanılabiliyor.
      const path = target.startsWith("xl/") ? target : `xl/${target}`
      if (entries.has(path)) return path
    }
  }

  const guess = [...entries.keys()]
    .filter((name) => /^xl\/worksheets\/sheet\d+\.xml$/.test(name))
    .sort()[0]
  if (!guess) throw new XlsxError("Çalışma kitabında sayfa yok.")
  return guess
}

function readSharedStrings(doc: Document) {
  return Array.from(doc.querySelectorAll("sst > si")).map(textOf)
}

/** `<t>` düğümlerinin birleşimi; fonetik okuma (`rPh`) dışarıda kalır. */
function textOf(node: Element | null) {
  if (!node) return ""
  return Array.from(node.querySelectorAll("t"))
    .filter((element) => !element.closest("rPh"))
    .map((element) => element.textContent ?? "")
    .join("")
}

function readGrid(sheet: Document, shared: string[]) {
  const grid: string[][] = []
  let truncated = false

  for (const row of Array.from(sheet.querySelectorAll("sheetData > row"))) {
    if (grid.length >= LIMITS.rows) {
      truncated = true
      break
    }

    const cells: string[] = []
    for (const cell of Array.from(row.querySelectorAll("c"))) {
      const reference = cell.getAttribute("r") ?? ""
      const column = reference ? columnIndex(reference) : cells.length
      if (column < 0 || column >= LIMITS.columns) continue
      while (cells.length < column) cells.push("")
      cells[column] = cellText(cell, shared)
    }

    grid.push(cells)
  }

  return { grid, truncated }
}

function cellText(cell: Element, shared: string[]) {
  const type = cell.getAttribute("t")

  if (type === "s") {
    const index = Number(cell.querySelector("v")?.textContent ?? "")
    return clean(shared[index] ?? "")
  }
  if (type === "inlineStr") {
    return clean(textOf(cell.querySelector("is")))
  }
  // Hata hücresi (#REF!, #NAME?) boş sayılır.
  if (type === "e") return ""

  return clean(cell.querySelector("v")?.textContent ?? "")
}

/** Denetim karakterleri ve yön değiştirme işaretleri atılır, uzunluk kırpılır. */
function clean(value: string) {
  return (
    value

      .replace(/[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f]/g, "")
      // Sağdan-sola geçişleri kimlik gizlemek için kullanılabiliyor.
      .replace(/[\u200e\u200f\u202a-\u202e\u2066-\u2069]/g, "")
      .trim()
      .slice(0, LIMITS.cellChars)
  )
}

/** "BC12" → 54. Sütun harfleri 26 tabanlıdır ama sıfırsızdır. */
function columnIndex(reference: string) {
  let value = 0
  for (const character of reference) {
    const code = character.charCodeAt(0)
    if (code < 65 || code > 90) break
    value = value * 26 + (code - 64)
  }
  return value - 1
}
