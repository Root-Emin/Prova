"use client"

import * as React from "react"
import { useRouter } from "next/navigation"
import { ShieldCheck } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { FormShell } from "@/components/prova/form-shell"
import { graphqlRequest } from "@/lib/graphql"

export default function LoginPage() {
  const router = useRouter()
  const [email, setEmail] = React.useState("")
  const [password, setPassword] = React.useState("")
  const [error, setError] = React.useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = React.useState(false)

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const trimmed = email.trim()
    if (!trimmed || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(trimmed)) {
      setError("Geçerli bir kullanıcı adı/e-posta girin")
      return
    }
    if (password.length < 8) {
      setError("Şifre en az 8 karakter olmalı")
      return
    }
    setError(null)
    setIsSubmitting(true)
    try {
      const response = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: trimmed, password }),
      })
      const payload = (await response.json()) as {
        email?: string
        name?: string
        message?: string
      }
      if (!response.ok) throw new Error(payload.message ?? "Giriş yapılamadı")

      window.localStorage.setItem(
        "prova:admin-session",
        JSON.stringify({ email: payload.email, name: payload.name, role: "admin" }),
      )

      // Backend adresi tanımlandığında web login'i gerçek GraphQL oturumuyla
      // tamamlanır. Vercel demo ortamında yönetim ekranı yerel taslakla çalışır.
      if (process.env.NEXT_PUBLIC_GRAPHQL_URL) {
        const data = await graphqlRequest<{
          login: { accessToken: string; refreshToken: string }
        }>(
          `mutation Login($input: PasswordLoginInput!) {
            login(input: $input) { accessToken refreshToken }
          }`,
          { input: { email: trimmed, password } },
        )
        window.localStorage.setItem("prova:access-token", data.login.accessToken)
        window.localStorage.setItem("prova:refresh-token", data.login.refreshToken)
      }

      const next = new URLSearchParams(window.location.search).get("next")
      router.replace(next || "/")
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : "Giriş yapılamadı",
      )
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <FormShell
      title="Yönetici girişi"
      description="Prova yönetim paneline erişmek için kurum yöneticisi hesabınızla giriş yapın."
      aside={
        <div className="flex items-start gap-3 rounded-lg border border-line-strong bg-tint px-4 py-3">
          <ShieldCheck size={20} className="mt-0.5 shrink-0 text-primary" aria-hidden />
          <div>
            <p className="text-sm font-medium text-foreground">Yalnızca yetkili yöneticiler</p>
            <p className="mt-0.5 text-sm leading-relaxed text-muted-foreground">
              Kullanıcı ekleme, cihaz ve eğitim ayarları bu alandan yönetilir.
            </p>
          </div>
        </div>
      }
      footer="Çalışan hesabı, doğrulama koduyla Prova Desktop uygulamasında kullanılır."
    >
      <form onSubmit={handleSubmit} className="space-y-5" noValidate>
        <div className="space-y-2">
          <Label htmlFor="login-email">
            Yönetici e-postası <span className="text-destructive">*</span>
          </Label>
          <Input
            id="login-email"
            type="email"
            required
            autoComplete="username"
            value={email}
            onChange={(event) => {
              setEmail(event.target.value)
              if (error) setError(null)
            }}
            aria-invalid={error ? true : undefined}
            aria-describedby={error ? "login-email-error" : undefined}
          />
          {error ? (
            <p id="login-email-error" className="text-sm text-destructive">
              {error}
            </p>
          ) : null}
        </div>
        <div className="space-y-2">
          <Label htmlFor="login-password">
            Şifre <span className="text-destructive">*</span>
          </Label>
          <Input
            id="login-password"
            type="password"
            required
            minLength={8}
            autoComplete="current-password"
            value={password}
            onChange={(event) => {
              setPassword(event.target.value)
              if (error) setError(null)
            }}
            aria-invalid={error ? true : undefined}
            aria-describedby={error ? "login-password-error" : undefined}
          />
        </div>
        <div className="pt-1">
          <Button
            type="submit"
            className="w-full"
            size="lg"
            disabled={isSubmitting}
          >
            {isSubmitting ? "Giriş yapılıyor…" : "Giriş yap"}
          </Button>
        </div>
      </form>
    </FormShell>
  )
}
