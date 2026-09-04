"use client"

import * as React from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { FormShell } from "@/components/prova/form-shell"

export default function LoginPage() {
  const router = useRouter()
  const [email, setEmail] = React.useState("")
  const [error, setError] = React.useState<string | null>(null)

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const trimmed = email.trim()
    if (!trimmed || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(trimmed)) {
      setError("Geçerli bir e-posta girin")
      return
    }
    setError(null)
    router.push("/verify")
  }

  return (
    <FormShell
      title="Giriş yap"
      description="Parola yok; e-postanıza tek kullanımlık kod gönderilir."
      footer={
        <>
          Hesabınız yok mu?{" "}
          <Link href="/register" className="text-primary underline underline-offset-4">
            Kayıt olun
          </Link>
        </>
      }
    >
      <form onSubmit={handleSubmit} className="space-y-5" noValidate>
        <div className="space-y-2">
          <Label htmlFor="login-email">
            Kurumsal e-posta <span className="text-destructive">*</span>
          </Label>
          <Input
            id="login-email"
            type="email"
            required
            autoComplete="email"
            value={email}
            onChange={(event) => {
              setEmail(event.target.value)
              if (error) setError(null)
            }}
            placeholder="ad.soyad@kurum.tr"
            aria-invalid={error ? true : undefined}
            aria-describedby={error ? "login-email-error" : undefined}
          />
          {error ? (
            <p id="login-email-error" className="text-sm text-destructive">
              {error}
            </p>
          ) : null}
        </div>
        <div className="pt-1">
          <Button type="submit" className="w-full" size="lg">
            Kod gönder
          </Button>
        </div>
      </form>
    </FormShell>
  )
}
