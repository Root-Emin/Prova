"use client"

import * as React from "react"
import { FileSpreadsheet, ShieldCheck, Upload } from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { cn } from "@/lib/utils"
import { XLSX_ACCEPT, XlsxError, readFirstSheet } from "@/lib/xlsx-import"

import { MappingError, mapSheet, type CandidateState, type ImportReport } from "./import-mapping"
import { userRoleLabel } from "./mock"
import { useUsers } from "./users-store"

const stateLabel: Record<CandidateState, string> = {
  new: "Eklenecek",
  existing: "Zaten kayıtlı",
  duplicate: "Tekrar",
  invalid: "Hatalı",
}

/**
 * Excel ile toplu kullanıcı tanımlama.
 *
 * Dosya sunucuya gitmez, tarayıcıda okunur ve önce önizlenir: hangi satırın
 * ekleneceği, hangisinin neden elendiği onaydan önce görünür. Kabul edilen tek
 * biçim .xlsx'tir; kural listesi ve gerekçeleri `@/lib/xlsx-import` başındadır.
 */
export function BulkImportDialog({
  departmentId,
  lockDepartment = false,
}: {
  departmentId: string
  /** Departman ekranından açıldığında hedef sabittir. */
  lockDepartment?: boolean
}) {
  const { departments, users, addUsers } = useUsers()
  const inputRef = React.useRef<HTMLInputElement>(null)

  const [open, setOpen] = React.useState(false)
  const [target, setTarget] = React.useState(departmentId)
  const [fileName, setFileName] = React.useState<string | null>(null)
  const [busy, setBusy] = React.useState(false)
  const [dragging, setDragging] = React.useState(false)
  const [error, setError] = React.useState<string | null>(null)
  const [report, setReport] = React.useState<ImportReport | null>(null)

  const departmentOptions = React.useMemo(
    () =>
      Object.fromEntries(
        departments.map((department) => [department.id, department.name])
      ),
    [departments]
  )

  function reset() {
    setFileName(null)
    setError(null)
    setReport(null)
    setBusy(false)
    setDragging(false)
    if (inputRef.current) inputRef.current.value = ""
  }

  function openChange(next: boolean) {
    setOpen(next)
    if (!next) reset()
    if (next) setTarget(departmentId)
  }

  async function handleFile(file: File | undefined) {
    if (!file) return
    setError(null)
    setReport(null)
    setFileName(file.name)
    setBusy(true)

    try {
      const sheet = await readFirstSheet(file)
      const taken = new Set(users.map((user) => user.email))
      setReport(mapSheet(sheet, taken))
    } catch (cause) {
      // Reddedilen dosyanın adı bırakılırsa kabul edilmiş gibi okunuyor.
      setFileName(null)
      // Beklenen ret gerekçeleri kullanıcıya aynen gösterilir; gerisi geneldir.
      setError(
        cause instanceof XlsxError || cause instanceof MappingError
          ? cause.message
          : "Dosya okunamadı."
      )
    } finally {
      setBusy(false)
      if (inputRef.current) inputRef.current.value = ""
    }
  }

  function confirm() {
    if (!report) return
    const added = addUsers(
      report.fresh.map(({ email, name, role }) => ({ email, name, role })),
      target
    )
    const department = departments.find((item) => item.id === target)
    toast.success(`${added} kullanıcı eklendi`, {
      description: `${department?.name ?? "Departman"} · giriş bekleniyor olarak işaretlendi.`,
    })
    openChange(false)
  }

  const counts = report
    ? {
        fresh: report.fresh.length,
        existing: report.candidates.filter((row) => row.state === "existing").length,
        rejected: report.candidates.filter(
          (row) => row.state === "invalid" || row.state === "duplicate"
        ).length,
      }
    : null

  return (
    <Dialog open={open} onOpenChange={openChange}>
      <DialogTrigger render={<Button variant="outline" size="lg" />}>
        <FileSpreadsheet aria-hidden />
        Toplu yükleme
      </DialogTrigger>

      <DialogContent className="sm:max-w-[680px]">
        <DialogHeader>
          <DialogTitle>Excel ile toplu kullanıcı yükleme</DialogTitle>
          <DialogDescription>
            Listedeki adresler departmana tanımlanır. Kullanıcı masaüstü
            uygulamasından kendi adresiyle giriş yapana kadar “giriş bekleniyor”
            durumunda kalır.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {!lockDepartment && (
            <div className="space-y-2">
              <Label>Hedef departman</Label>
              <Select
                items={departmentOptions}
                value={target}
                onValueChange={(value) => setTarget(value as string)}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {departments.map((department) => (
                    <SelectItem key={department.id} value={department.id}>
                      {department.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          {report ? (
            <ImportPreview report={report} fileName={fileName} />
          ) : (
            <DropZone
              accept={XLSX_ACCEPT}
              busy={busy}
              dragging={dragging}
              fileName={fileName}
              inputRef={inputRef}
              onDraggingChange={setDragging}
              onFile={handleFile}
            />
          )}

          {error && (
            <p
              role="alert"
              className="rounded-lg border border-line-strong bg-tint px-3 py-2 text-sm"
            >
              {error}
            </p>
          )}

          {!report && <Rules />}
        </div>

        <DialogFooter>
          {report ? (
            <>
              <Button variant="outline" type="button" onClick={reset}>
                Başka dosya seç
              </Button>
              <Button
                type="button"
                onClick={confirm}
                disabled={counts?.fresh === 0}
              >
                {counts?.fresh === 0
                  ? "Eklenecek kayıt yok"
                  : `${counts?.fresh} kullanıcı ekle`}
              </Button>
            </>
          ) : (
            <DialogClose render={<Button variant="outline" type="button" />}>
              Vazgeç
            </DialogClose>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function DropZone({
  accept,
  busy,
  dragging,
  fileName,
  inputRef,
  onDraggingChange,
  onFile,
}: {
  accept: string
  busy: boolean
  dragging: boolean
  fileName: string | null
  inputRef: React.RefObject<HTMLInputElement | null>
  onDraggingChange: (dragging: boolean) => void
  onFile: (file: File | undefined) => void
}) {
  return (
    <div
      onDragOver={(event) => {
        event.preventDefault()
        onDraggingChange(true)
      }}
      onDragLeave={() => onDraggingChange(false)}
      onDrop={(event) => {
        event.preventDefault()
        onDraggingChange(false)
        // Birden çok dosya bırakılsa da yalnızca ilki okunur.
        onFile(event.dataTransfer.files?.[0])
      }}
      className={cn(
        "rounded-lg border border-dashed border-line-strong px-4 py-6 text-center transition-colors",
        dragging && "bg-muted"
      )}
    >
      <Upload size={20} className="mx-auto text-primary" aria-hidden />
      <p className="mt-2 text-sm font-medium">
        {busy
          ? "Dosya okunuyor…"
          : fileName
            ? fileName
            : "Dosyayı buraya bırakın"}
      </p>
      <p className="prova-meta mt-1 normal-case">
        Yalnızca .xlsx · en fazla 2 MB · ilk sayfa okunur
      </p>

      <input
        ref={inputRef}
        type="file"
        accept={accept}
        className="sr-only"
        onChange={(event) => onFile(event.target.files?.[0])}
      />
      <Button
        type="button"
        variant="outline"
        size="sm"
        className="mt-3"
        disabled={busy}
        onClick={() => inputRef.current?.click()}
      >
        Dosya seç
      </Button>
    </div>
  )
}

/** Beklenen sütunlar ve dosyanın neden reddedilebileceği. */
function Rules() {
  return (
    <div className="rounded-lg border border-border bg-muted px-3 py-2.5">
      <p className="prova-meta uppercase">Beklenen sütunlar</p>
      <p className="mt-1 text-sm">
        <span className="font-medium">E-posta</span> zorunlu.{" "}
        <span className="font-medium">Ad soyad</span> ve{" "}
        <span className="font-medium">Rol</span> varsa okunur, yoksa ad adresten
        türetilir ve rol “{userRoleLabel.employee}” kabul edilir. Başlık satırı
        yoksa A sütunu adres sayılır.
      </p>
      <p className="mt-2 flex items-start gap-2 text-sm text-muted-foreground">
        <ShieldCheck size={16} className="mt-0.5 shrink-0" aria-hidden />
        <span>
          Dosya tarayıcıdan çıkmaz. Uzantı, tür ve içerik imzası ayrı ayrı
          doğrulanır; makro, dış veri bağlantısı veya gömülü nesne taşıyan
          dosyalar açılmadan reddedilir.
        </span>
      </p>
    </div>
  )
}

function ImportPreview({
  report,
  fileName,
}: {
  report: ImportReport
  fileName: string | null
}) {
  const fresh = report.fresh.length
  const existing = report.candidates.filter((row) => row.state === "existing").length
  const rejected = report.candidates.length - fresh - existing

  return (
    <div className="space-y-3">
      <dl className="grid grid-cols-3 gap-2">
        <Tally label="Eklenecek" value={fresh} />
        <Tally label="Zaten kayıtlı" value={existing} />
        <Tally label="Elenen satır" value={rejected} />
      </dl>

      <div className="max-h-[260px] min-w-0 overflow-y-auto rounded-lg border border-border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[56px]">Satır</TableHead>
              <TableHead>E-posta</TableHead>
              <TableHead className="w-[150px]">Ad soyad</TableHead>
              <TableHead className="w-[130px]">Durum</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {report.candidates.map((candidate) => (
              <TableRow key={`${candidate.row}-${candidate.email}`}>
                <TableCell className="prova-meta">{candidate.row}</TableCell>
                <TableCell
                  className={cn(
                    "font-mono text-xs",
                    candidate.state !== "new" && "text-muted-foreground"
                  )}
                >
                  {candidate.email || "—"}
                </TableCell>
                <TableCell
                  className={cn(
                    candidate.state !== "new" && "text-muted-foreground"
                  )}
                >
                  {candidate.name || "—"}
                </TableCell>
                <TableCell
                  className={cn(
                    "prova-meta normal-case",
                    candidate.state === "new" && "text-foreground"
                  )}
                >
                  {candidate.note ?? stateLabel[candidate.state]}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <p className="prova-meta normal-case">
        {fileName ? `${fileName} · ` : ""}“{report.sheetName}” sayfası okundu
        {report.headerRow === null ? " · başlık satırı yok" : ""}
        {report.truncated ? " · satır sınırına takıldı, liste kesildi" : ""}
      </p>
    </div>
  )
}

function Tally({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg border border-border bg-card px-3 py-2">
      <dt className="prova-meta uppercase">{label}</dt>
      <dd className="font-heading text-xl leading-none text-primary">{value}</dd>
    </div>
  )
}
