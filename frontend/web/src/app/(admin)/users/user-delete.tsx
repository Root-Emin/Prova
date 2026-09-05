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

import type { User } from "./mock"
import { useUsers } from "./users-store"

/**
 * Kişiyi kurumdan çıkarma.
 *
 * Pasifleştirme ile silme farklı işlerdir: pasif kullanıcı listede kalır ve
 * geri açılabilir, silinen kişi kayıttan düşer. Bu yüzden onay metni neyin
 * gittiğini, neyin denetim için kaldığını tek tek söyler.
 */
export function UserDelete({
  user,
  departmentName,
  onDeleted,
  iconOnly = false,
  className,
}: {
  user: User
  departmentName?: string
  onDeleted?: (user: User) => void
  /** Tablo satırında yalnızca ikon; ayrıntı ekranında metinli düğme. */
  iconOnly?: boolean
  className?: string
}) {
  const { removeUser } = useUsers()
  const [open, setOpen] = React.useState(false)

  function confirm() {
    const removed = removeUser(user.id)
    setOpen(false)
    if (!removed) return
    toast.success(`${removed.name} kurumdan çıkarıldı`, {
      description: "İşlem denetim kaydına yazıldı.",
    })
    onDeleted?.(removed)
  }

  return (
    <AlertDialog open={open} onOpenChange={setOpen}>
      <AlertDialogTrigger
        render={
          <Button
            variant={iconOnly ? "ghost" : "outline"}
            size={iconOnly ? "sm" : "default"}
            className={className}
          />
        }
        aria-label={`${user.name} kişisini sil`}
      >
        <Trash2 aria-hidden />
        {iconOnly ? null : "Kullanıcıyı sil"}
      </AlertDialogTrigger>

      <AlertDialogContent className="sm:max-w-md">
        <AlertDialogHeader>
          <AlertDialogTitle>{user.name} silinsin mi?</AlertDialogTitle>
          <AlertDialogDescription>
            {user.email}
            {departmentName ? ` · ${departmentName}` : ""} kaydı kalıcı olarak
            kaldırılır: kayıtlı cihazları ve bekleyen atamaları da gider.
            Tamamlanmış oturum ve sertifika kayıtları denetim için durur.
            Kişinin yalnızca girişini kapatmak istiyorsanız silmek yerine
            pasifleştirin.
          </AlertDialogDescription>
        </AlertDialogHeader>

        <AlertDialogFooter>
          <AlertDialogCancel>Vazgeç</AlertDialogCancel>
          <Button onClick={confirm}>Kullanıcıyı sil</Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
