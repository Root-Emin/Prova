"use client"

import * as React from "react"
import { Trash2 } from "lucide-react"
import { toast } from "sonner"

import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

import type { Department } from "./mock"
import { useUsers } from "./users-store"

/**
 * Departman kapatma.
 *
 * Departmanın personeli varsa kapatma tek başına yapılmaz: kişiler önce başka
 * bir departmana taşınır. Kişi silmek ayrı ve bilinçli bir karardır, departman
 * kapatmanın sessiz yan etkisi olamaz.
 */
export function DepartmentDelete({
  department,
  memberCount,
  onDeleted,
  className,
}: {
  department: Department
  memberCount: number
  onDeleted?: () => void
  className?: string
}) {
  const { departments, removeDepartment } = useUsers()

  const others = departments.filter((item) => item.id !== department.id)
  const [open, setOpen] = React.useState(false)
  const [target, setTarget] = React.useState(others[0]?.id ?? "")

  const targetOptions = React.useMemo(
    () => Object.fromEntries(others.map((item) => [item.id, item.name])),
    [others]
  )

  // Kurumda tek departman kaldıysa taşınacak yer yok; kapatma da yapılmaz.
  const blocked = others.length === 0
  const needsMove = memberCount > 0

  function openChange(next: boolean) {
    setOpen(next)
    if (next) setTarget(others[0]?.id ?? "")
  }

  function confirm() {
    removeDepartment(department.id, needsMove ? target : undefined)
    setOpen(false)
    toast.success(`${department.name} kapatıldı`, {
      description: needsMove
        ? `${memberCount} kişi ${targetOptions[target]} departmanına taşındı.`
        : "Departmanda kayıtlı kişi yoktu.",
    })
    onDeleted?.()
  }

  return (
    <AlertDialog open={open} onOpenChange={openChange}>
      <AlertDialogTrigger
        render={<Button variant="ghost" size="sm" className={className} />}
        aria-label={`${department.name} departmanını kapat`}
      >
        <Trash2 aria-hidden />
        Kapat
      </AlertDialogTrigger>

      <AlertDialogContent className="sm:max-w-md">
        <AlertDialogHeader>
          <AlertDialogTitle>
            {department.name} departmanı kapatılsın mı?
          </AlertDialogTitle>
          <AlertDialogDescription>
            {blocked
              ? "Kurumda başka departman yok. Son departman kapatılamaz; önce yeni bir departman açın."
              : needsMove
                ? `Departmanda ${memberCount} kişi var. Kapatmadan önce hepsi seçtiğiniz departmana taşınır; kimse silinmez.`
                : "Departmanda kayıtlı kişi yok, doğrudan kapatılabilir."}
          </AlertDialogDescription>
        </AlertDialogHeader>

        {!blocked && needsMove && (
          <div className="space-y-2">
            <Label>Personel şu departmana taşınsın</Label>
            <Select
              items={targetOptions}
              value={target}
              onValueChange={(value) => setTarget(value as string)}
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {others.map((item) => (
                  <SelectItem key={item.id} value={item.id}>
                    {item.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}

        <AlertDialogFooter>
          <AlertDialogCancel>Vazgeç</AlertDialogCancel>
          <Button onClick={confirm} disabled={blocked}>
            {needsMove ? "Taşı ve kapat" : "Departmanı kapat"}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
