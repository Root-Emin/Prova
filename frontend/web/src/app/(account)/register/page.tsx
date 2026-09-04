"use client"

import * as React from "react"
import Link from "next/link"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { FormShell } from "@/components/prova/form-shell"
import { PrivacyNotice } from "@/components/prova/privacy-notice"

export default function RegisterPage() {
  const [name, setAd] = React.useState("")
  const [email, setEmail] = React.useState("")

  return (
    <FormShell
      title="Hesap oluştur"
      description="Kurumsal e-postanıza doğrulama kodu göndeririz."
      aside={<PrivacyNotice />}
      action={
        <Button nativeButton={false} className="w-full" size="lg" render={<Link href="/verify" />}>
          Doğrulama kodu gönder
        </Button>
      }
      footer={
        <>
          Hesabınız var mı?{" "}
          <Link href="/login" className="text-primary underline underline-offset-4">
            Giriş yapın
          </Link>
        </>
      }
    >
      <div className="space-y-2">
        <Label htmlFor="register-name">Ad soyad</Label>
        <Input
          id="register-name"
          value={name}
          onChange={(event) => setAd(event.target.value)}
          placeholder="Ad Soyad"
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="register-email">Kurumsal e-posta</Label>
        <Input
          id="register-email"
          type="email"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
          placeholder="ad.soyad@kurum.tr"
        />
      </div>
    </FormShell>
  )
}
