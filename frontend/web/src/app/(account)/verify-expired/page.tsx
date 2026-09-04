"use client"

import Link from "next/link"
import { Clock } from "lucide-react"

import { Button } from "@/components/ui/button"
import { FormShell } from "@/components/prova/form-shell"

export default function VerifyExpiredPage() {
  return (
    <FormShell
      title="Bağlantının süresi dolmuş"
      description="Doğrulama bağlantıları 10 dakika geçerlidir ve tek kullanımlıktır."
      action={
        <Button
          nativeButton={false}
          className="w-full"
          size="lg"
          render={<Link href="/login" />}
        >
          Yeni kod iste
        </Button>
      }
      footer="Sorun sürerse kurum yöneticinize başvurun."
    >
      <div className="flex items-start gap-3 rounded-lg border border-line-strong bg-muted px-4 py-3">
        <Clock size={20} className="mt-0.5 shrink-0 text-primary" aria-hidden />
        <p className="text-sm leading-relaxed text-foreground">
          Bağlantı ya süresini doldurdu ya da daha önce kullanıldı. Güvenlik
          gereği aynı bağlantı ikinci kez çalışmaz.
        </p>
      </div>
    </FormShell>
  )
}
