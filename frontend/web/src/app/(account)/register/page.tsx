"use client"

import * as React from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { FormShell } from "@/components/prova/form-shell"
import { PrivacyNotice } from "@/components/prova/privacy-notice"
import { accountFlowStorage, graphqlRequest } from "@/lib/graphql"

export default function RegisterPage() {
  const router = useRouter()
  const [name, setAd] = React.useState("")
  const [email, setEmail] = React.useState("")
  const [errors, setErrors] = React.useState<{
    name?: string
    email?: string
  }>({})
  const [submitError, setSubmitError] = React.useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = React.useState(false)

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const next: { name?: string; email?: string } = {}
    const trimmedName = name.trim()
    const trimmedEmail = email.trim()

    if (!trimmedName) {
      next.name = "Ad soyad gerekli"
    }
    if (!trimmedEmail || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(trimmedEmail)) {
      next.email = "Geçerli bir e-posta girin"
    }

    if (next.name || next.email) {
      setErrors(next)
      return
    }

    setErrors({})
    setSubmitError(null)
    setIsSubmitting(true)
    const nameParts = trimmedName.split(/\s+/)
    const firstName = nameParts.shift() ?? ""
    const lastName = nameParts.join(" ")
    try {
      const data = await graphqlRequest<{
        register: { email: string; resendAfterSeconds: number }
      }>(
        `mutation Register($input: RegisterInput!) {
        register(input: $input) { email resendAfterSeconds }
      }`,
        { input: { email: trimmedEmail, firstName, lastName } },
      )
      window.sessionStorage.setItem(
        accountFlowStorage.email,
        data.register.email,
      )
      window.sessionStorage.setItem(
        accountFlowStorage.mode,
        "email-verification",
      )
      window.sessionStorage.setItem(
        accountFlowStorage.resendAfter,
        String(data.register.resendAfterSeconds),
      )
      router.push("/verify")
    } catch (error) {
      setSubmitError(
        error instanceof Error ? error.message : "Kayıt tamamlanamadı",
      )
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <FormShell
      title="Hesap oluştur"
      description="Kurumsal e-postanıza doğrulama kodu göndeririz."
      aside={<PrivacyNotice />}
      footer={
        <>
          Hesabınız var mı?{" "}
          <Link
            href="/login"
            className="text-primary underline underline-offset-4"
          >
            Giriş yapın
          </Link>
        </>
      }
    >
      <form onSubmit={handleSubmit} className="space-y-5" noValidate>
        <div className="space-y-2">
          <Label htmlFor="register-name">
            Ad soyad <span className="text-destructive">*</span>
          </Label>
          <Input
            id="register-name"
            required
            autoComplete="name"
            value={name}
            onChange={(event) => {
              setAd(event.target.value)
              setSubmitError(null)
              if (errors.name)
                setErrors((prev) => ({ ...prev, name: undefined }))
            }}
            placeholder="Ad Soyad"
            aria-invalid={errors.name ? true : undefined}
            aria-describedby={errors.name ? "register-name-error" : undefined}
          />
          {errors.name ? (
            <p id="register-name-error" className="text-sm text-destructive">
              {errors.name}
            </p>
          ) : null}
        </div>
        <div className="space-y-2">
          <Label htmlFor="register-email">
            Kurumsal e-posta <span className="text-destructive">*</span>
          </Label>
          <Input
            id="register-email"
            type="email"
            required
            autoComplete="email"
            value={email}
            onChange={(event) => {
              setEmail(event.target.value)
              setSubmitError(null)
              if (errors.email)
                setErrors((prev) => ({ ...prev, email: undefined }))
            }}
            placeholder="ad.soyad@kurum.tr"
            aria-invalid={errors.email ? true : undefined}
            aria-describedby={errors.email ? "register-email-error" : undefined}
          />
          {errors.email ? (
            <p id="register-email-error" className="text-sm text-destructive">
              {errors.email}
            </p>
          ) : null}
        </div>
        {submitError ? (
          <p className="text-sm text-destructive" role="alert">
            {submitError}
          </p>
        ) : null}
        <div className="pt-1">
          <Button
            type="submit"
            className="w-full"
            size="lg"
            disabled={isSubmitting}
          >
            {isSubmitting ? "Gönderiliyor…" : "Doğrulama kodu gönder"}
          </Button>
        </div>
      </form>
    </FormShell>
  )
}
