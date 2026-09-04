"use client"

import * as React from "react"
import Link from "next/link"
import { Laptop } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { FormShell } from "@/components/prova/form-shell"
import { useDeviceState } from "@/hooks/use-device-state"
import { loginInfo } from "./mock"

/** Parolasız hat: adres girilir, altı haneli kod bu adrese gider. */
function emailLooksValid(value: string) {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value.trim())
}

export default function LoginPage() {
  const device = useDeviceState()
  const [email, setEmail] = React.useState("")

  const ok = emailLooksValid(email)

  return (
    <FormShell
      title="Giriş yap"
      description="Parola yok; kurumsal e-postanıza tek kullanımlık kod gönderilir."
      action={
        <Button
          nativeButton={ok ? false : undefined}
          className="w-full"
          size="lg"
          disabled={!ok}
          render={ok ? <Link href="/verify" /> : undefined}
        >
          Kod gönder
        </Button>
      }
      footer="Hesabınız yoksa ilk girişte kurumsal adresinizle oluşturulur."
    >
      <div className="space-y-2">
        <Label htmlFor="login-email">Kurumsal e-posta</Label>
        <Input
          id="login-email"
          type="email"
          inputMode="email"
          autoComplete="email"
          autoFocus
          value={email}
          onChange={(event) => setEmail(event.target.value)}
          placeholder="ad.soyad@kurum.tr"
        />
      </div>

      <div className="flex items-start gap-3 rounded-lg border border-border bg-muted px-4 py-3">
        <Laptop size={20} className="mt-0.5 shrink-0 text-primary" aria-hidden />
        <div>
          <p className="text-sm leading-relaxed text-foreground">
            Giriş tamamlandığı anda bu cihaz hesabınıza bağlanır. Sınav
            yalnızca kayıtlı cihazdan verilebilir.
          </p>
          <p className="prova-meta mt-1 normal-case">
            {device.deviceName ?? "Bilinmeyen cihaz"} · {device.os} · parmak izi{" "}
            <span className="font-mono">{loginInfo.fingerprint}</span>
          </p>
        </div>
      </div>
    </FormShell>
  )
}
