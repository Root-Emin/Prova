"use client"

import * as React from "react"
import Link from "next/link"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { FormShell } from "@/components/prova/form-shell"

export default function LoginPage() {
  const [email, setEmail] = React.useState("")

  return (
    <FormShell
      title="Giriş yap"
      description="Parola yok; e-postanıza tek kullanımlık kod gönderilir."
      action={
        <Button nativeButton={false} className="w-full" size="lg" render={<Link href="/verify" />}>
          Kod gönder
        </Button>
      }
      footer={
        <>
          Hesabınız yok mu?{" "}
          <Link href="/register" className="text-primary underline underline-offset-4">
            Kayıt olun
          </Link>
        </>
      }
    >
      <div className="space-y-2">
        <Label htmlFor="login-email">Kurumsal e-posta</Label>
        <Input
          id="login-email"
          type="email"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
          placeholder="ad.soyad@kurum.tr"
        />
      </div>
    </FormShell>
  )
}
