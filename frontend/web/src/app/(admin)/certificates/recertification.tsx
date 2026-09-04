"use client"

import * as React from "react"
import { RefreshCw } from "lucide-react"
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
import { publishedVersions, type Certificate } from "./mock"

/**
 * When regulation changes and the rubric moves to a new version, everyone
 * certified against the old version is bulk-enrolled for recertification.
 */
export function Recertification({
  affected,
  onStart,
}: {
  affected: Certificate[]
  onStart: () => void
}) {
  const [open, setOpen] = React.useState(false)

  if (affected.length === 0) return null

  const scenario = affected[0].scenario
  const newVersion = publishedVersions[scenario]

  return (
    <section className="flex items-start justify-between gap-6 rounded-lg border border-line-strong bg-tint px-4 py-3">
      <div>
        <p className="font-heading text-sm font-semibold text-primary">
          {affected.length} sertifika eski rubrik sürümüyle verildi
        </p>
        <p className="mt-1 max-w-[620px] text-sm leading-relaxed text-foreground">
          “{scenario}” senaryosunun {newVersion} sürümü yayında. Daha eski sürümle
          sertifikalanan personelin sertifikası mevzuat değişikliği nedeniyle
          yenilenmelidir.
        </p>
      </div>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogTrigger render={<Button className="shrink-0" />}>
          <RefreshCw aria-hidden />
          Toplu yenileme başlat
        </DialogTrigger>
        <DialogContent className="sm:max-w-[480px]">
          <DialogHeader>
            <DialogTitle>Toplu yeniden sertifikasyon</DialogTitle>
            <DialogDescription>
              Aşağıdaki personele {newVersion} sürümüyle created oturum atanır.
              Mevcut sertifikaları created oturum tamamlanana kadar geçerli kalır.
            </DialogDescription>
          </DialogHeader>

          <ul className="divide-y divide-border py-4">
            {affected.map((certificate) => (
              <li
                key={certificate.id}
                className="flex items-center justify-between py-2"
              >
                <span className="text-sm">{certificate.employee}</span>
                <span className="prova-meta normal-case">
                  {certificate.id} · {certificate.rubricVersion} → {newVersion}
                </span>
              </li>
            ))}
          </ul>

          <DialogFooter>
            <DialogClose render={<Button variant="outline" type="button" />}>
              Vazgeç
            </DialogClose>
            <Button
              onClick={() => {
                onStart()
                setOpen(false)
                toast.success(
                  `${affected.length} kişiye yeniden sertifikasyon oturumu atandı`,
                  { description: "Son tarih: 30 gün." }
                )
              }}
            >
              Atamaları oluştur
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </section>
  )
}
