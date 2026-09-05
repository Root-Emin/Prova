"use client"

import * as React from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { FormShell } from "@/components/prova/form-shell"
import { accountFlowStorage, graphqlRequest } from "@/lib/graphql"

export default function LoginPage() {
  const router = useRouter()
  const [email, setEmail] = React.useState("")
  const [error, setError] = React.useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = React.useState(false)

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const trimmed = email.trim()
    if (!trimmed || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(trimmed)) {
      setError("Geçerli bir e-posta girin")
      return
    }
    setError(null)
    setIsSubmitting(true)
    try {
      const data = await graphqlRequest<{
        requestLoginCode: { resendAfterSeconds: number }
      }>(
        `mutation RequestLoginCode($input: RequestLoginCodeInput!) {
        requestLoginCode(input: $input) { resendAfterSeconds }
      }`,
        { input: { email: trimmed } },
      )
      window.sessionStorage.setItem(
        accountFlowStorage.email,
        trimmed.toLowerCase(),
      )
      window.sessionStorage.setItem(accountFlowStorage.mode, "login")
      window.sessionStorage.setItem(
        accountFlowStorage.resendAfter,
        String(data.requestLoginCode.resendAfterSeconds),
      )
      router.push("/verify")
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : "Kod gönderilemedi",
      )
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <FormShell
      title="Giriş yap"
      description="Parola yok; e-postanıza tek kullanımlık kod gönderilir."
      footer={
        <>
          Hesabınız yok mu?{" "}
          <Link
            href="/register"
            className="text-primary underline underline-offset-4"
          >
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
          <Button
            type="submit"
            className="w-full"
            size="lg"
            disabled={isSubmitting}
          >
            {isSubmitting ? "Gönderiliyor…" : "Kod gönder"}
          </Button>
        </div>
      </form>
    </FormShell>
  )
}
