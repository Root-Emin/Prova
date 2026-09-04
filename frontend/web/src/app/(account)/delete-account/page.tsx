"use client"

import * as React from "react"

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { FormShell } from "@/components/prova/form-shell"
import { accountToDelete } from "./mock"

export default function DeleteAccountPage() {
  const [open, setOpen] = React.useState(false)
  const [step, setStep] = React.useState<1 | 2>(1)
  const [verification, setVerification] = React.useState("")

  const secondStepDone = verification.trim() === accountToDelete.confirmWord

  function close(nextOpen: boolean) {
    setOpen(nextOpen)
    if (!nextOpen) {
      setStep(1)
      setVerification("")
    }
  }

  return (
    <FormShell
      title="Hesabı sil"
      description="Silme talebi geri alınamaz ve 30 gün içinde tamamlanır."
      action={
        <AlertDialog open={open} onOpenChange={close}>
          <AlertDialogTrigger render={<Button className="w-full" size="lg" />}>
            Hesabı silme talebi oluştur
          </AlertDialogTrigger>
          <AlertDialogContent>
            {step === 1 ? (
              <>
                <AlertDialogHeader>
                  <AlertDialogTitle>Emin misiniz?</AlertDialogTitle>
                  <AlertDialogDescription>
                    {accountToDelete.email} hesabı ve bağlı tüm oturum verileri
                    silinir. Bu işlem geri alınamaz.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Vazgeç</AlertDialogCancel>
                  <Button onClick={() => setStep(2)}>Devam et</Button>
                </AlertDialogFooter>
              </>
            ) : (
              <>
                <AlertDialogHeader>
                  <AlertDialogTitle>Son onay</AlertDialogTitle>
                  <AlertDialogDescription>
                    Onaylamak için kutuya {accountToDelete.confirmWord} yazın.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <div className="space-y-2 py-4">
                  <Label htmlFor="delete-confirm">Onay</Label>
                  <Input
                    id="delete-confirm"
                    value={verification}
                    onChange={(event) => setVerification(event.target.value)}
                    placeholder={accountToDelete.confirmWord}
                    autoFocus
                  />
                </div>
                <AlertDialogFooter>
                  <AlertDialogCancel>Vazgeç</AlertDialogCancel>
                  <AlertDialogAction disabled={!secondStepDone}>
                    Hesabı sil
                  </AlertDialogAction>
                </AlertDialogFooter>
              </>
            )}
          </AlertDialogContent>
        </AlertDialog>
      }
      footer="Talebiniz denetim kaydına işlenir."
    >
      <div className="space-y-3">
        <div>
          <p className="prova-meta uppercase">Silinecek</p>
          <ul className="mt-1 space-y-1">
            {accountToDelete.deleted.map((clause) => (
              <li key={clause} className="text-sm text-foreground">
                {clause}
              </li>
            ))}
          </ul>
        </div>
        <div className="border-t border-border pt-3">
          <p className="prova-meta uppercase">Saklanmaya devam eden</p>
          <ul className="mt-1 space-y-1">
            {accountToDelete.retained.map((clause) => (
              <li key={clause} className="text-sm text-muted-foreground">
                {clause}
              </li>
            ))}
          </ul>
        </div>
      </div>
    </FormShell>
  )
}
