"use client"

import * as React from "react"
import { useRouter } from "next/navigation"

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
      router.push("/")
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
      title="Giriş yap"
      description="Test hesabınızın kullanıcı adı ve şifresiyle giriş yapın."
      footer="Demo: yonetici@prova.local / SecurePass123!"
    >
      <form onSubmit={handleSubmit} className="space-y-5" noValidate>
        <div className="space-y-2">
          <Label htmlFor="login-email">
            Kullanıcı adı / e-posta <span className="text-destructive">*</span>
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
            placeholder="yonetici@prova.local"
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
            placeholder="SecurePass123!"
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
