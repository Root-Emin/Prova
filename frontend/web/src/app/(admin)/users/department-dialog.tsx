"use client"

import * as React from "react"
import { Building2 } from "lucide-react"
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
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

import { useUsers } from "./users-store"

/** Ad "Uyum ve Risk" ise kod önerisi "UYR"; yönetici isterse değiştirir. */
function codeFrom(name: string) {
  const words = name.trim().split(/\s+/).filter((word) => word.length > 2)
  if (words.length === 0) return ""
  return words
    .slice(0, 3)
    .map((word) => word[0])
    .join("")
    .toLocaleUpperCase("tr-TR")
}

export function DepartmentDialog() {
  const { addDepartment } = useUsers()

  const [open, setOpen] = React.useState(false)
  const [name, setName] = React.useState("")
  const [code, setCode] = React.useState("")
  const [lead, setLead] = React.useState("")
  const [site, setSite] = React.useState("")

  function openChange(next: boolean) {
    setOpen(next)
    if (!next) {
      setName("")
      setCode("")
      setLead("")
      setSite("")
    }
  }

  function submit(event: React.FormEvent) {
    event.preventDefault()
    const department = addDepartment({
      name: name.trim(),
      code: (code.trim() || codeFrom(name)).toLocaleUpperCase("tr-TR"),
      lead: lead.trim() || "Tanımlanmadı",
      site: site.trim() || "Tanımlanmadı",
    })
    toast.success(`${department.name} departmanı açıldı`, {
      description: "Personeli Excel ile toplu yükleyebilir veya tek tek davet edebilirsiniz.",
    })
    openChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={openChange}>
      <DialogTrigger render={<Button variant="outline" size="lg" />}>
        <Building2 aria-hidden />
        Departman ekle
      </DialogTrigger>
      <DialogContent className="sm:max-w-[440px]">
        <form onSubmit={submit}>
          <DialogHeader>
            <DialogTitle>Departman ekle</DialogTitle>
            <DialogDescription>
              Her kullanıcı tek departmana bağlıdır. Departman açıldıktan sonra
              personeli içeriden tanımlanır.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-5">
            <div className="space-y-2">
              <Label htmlFor="departman-ad">Departman adı</Label>
              <Input
                id="departman-ad"
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder="Uyum ve Risk"
                required
              />
            </div>

            <div className="prova-alan-izgara">
              <div className="space-y-2">
                <Label htmlFor="departman-kod">Kısa kod</Label>
                <Input
                  id="departman-kod"
                  value={code}
                  onChange={(event) => setCode(event.target.value)}
                  placeholder={codeFrom(name) || "UYM"}
                  maxLength={5}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="departman-konum">Konum</Label>
                <Input
                  id="departman-konum"
                  value={site}
                  onChange={(event) => setSite(event.target.value)}
                  placeholder="Ankara · Bölge"
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="departman-yonetici">Departman yöneticisi</Label>
              <Input
                id="departman-yonetici"
                value={lead}
                onChange={(event) => setLead(event.target.value)}
                placeholder="Ad Soyad"
              />
            </div>

            <p className="prova-meta normal-case">
              Kısa kod listelerde adın yerine görünür; boş bırakılırsa addan
              türetilir.
            </p>
          </div>

          <DialogFooter>
            <DialogClose render={<Button variant="outline" type="button" />}>
              Vazgeç
            </DialogClose>
            <Button type="submit" disabled={name.trim().length === 0}>
              Departmanı aç
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
