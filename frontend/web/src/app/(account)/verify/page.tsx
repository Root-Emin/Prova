"use client"

import * as React from "react"
import { useRouter } from "next/navigation"

import { Button } from "@/components/ui/button"
import {
  InputOTP,
  InputOTPGroup,
  InputOTPSlot,
} from "@/components/ui/input-otp"
import { FormShell } from "@/components/prova/form-shell"
import { accountFlowStorage, graphqlRequest } from "@/lib/graphql"

const codeLength = 6
type FlowMode = "email-verification" | "login"

export default function VerifyPage() {
  const router = useRouter()
  const [code, setCode] = React.useState("")
  const [error, setError] = React.useState<string | null>(null)
  const [email, setEmail] = React.useState("")
  const [mode, setMode] = React.useState<FlowMode>("email-verification")
  const [isSubmitting, setIsSubmitting] = React.useState(false)
  const [isResending, setIsResending] = React.useState(false)
  const [remainingSeconds, setRemainingSeconds] = React.useState(60)

  React.useEffect(() => {
    const timer = window.setTimeout(() => {
      const storedEmail =
        window.sessionStorage.getItem(accountFlowStorage.email) ?? ""
      const storedMode = window.sessionStorage.getItem(accountFlowStorage.mode)
      const storedCooldown = Number(
        window.sessionStorage.getItem(accountFlowStorage.resendAfter) ?? "60",
      )
      setEmail(storedEmail)
      setMode(storedMode === "login" ? "login" : "email-verification")
      setRemainingSeconds(Number.isFinite(storedCooldown) ? storedCooldown : 60)
    }, 0)
    return () => window.clearTimeout(timer)
  }, [])

  React.useEffect(() => {
    if (remainingSeconds <= 0) return
    const ticker = window.setInterval(() => {
      setRemainingSeconds((prev) => prev - 1)
    }, 1000)
    return () => window.clearInterval(ticker)
  }, [remainingSeconds])

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!email) {
      setError("E-posta bilgisi bulunamadı. Lütfen işlemi baştan başlatın.")
      return
    }
    if (code.length !== codeLength || !/^\d{6}$/.test(code)) {
      setError("6 haneli kodu girin")
      return
    }
    setError(null)
    setIsSubmitting(true)
    try {
      if (mode === "email-verification") {
        await graphqlRequest<{ verifyEmail: { verified: boolean } }>(
          `mutation VerifyEmail($input: VerifyEmailInput!) {
          verifyEmail(input: $input) { verified }
        }`,
          { input: { email, code } },
        )
        window.sessionStorage.removeItem(accountFlowStorage.email)
        window.sessionStorage.removeItem(accountFlowStorage.mode)
        window.sessionStorage.removeItem(accountFlowStorage.resendAfter)
        router.push("/verify-result")
      } else {
        const data = await graphqlRequest<{
          verifyLoginCode: { accessToken: string; refreshToken: string }
        }>(
          `mutation VerifyLoginCode($input: VerifyLoginCodeInput!) {
          verifyLoginCode(input: $input) { accessToken refreshToken }
        }`,
          { input: { email, code } },
        )
        window.localStorage.setItem(
          "prova:access-token",
          data.verifyLoginCode.accessToken,
        )
        window.localStorage.setItem(
          "prova:refresh-token",
          data.verifyLoginCode.refreshToken,
        )
        router.push("/")
      }
    } catch (verifyError) {
      setError(
        verifyError instanceof Error
          ? verifyError.message
          : "Kod doğrulanamadı",
      )
    } finally {
      setIsSubmitting(false)
    }
  }

  async function handleResend() {
    if (!email || isResending) return
    setError(null)
    setIsResending(true)
    try {
      const data =
        mode === "email-verification"
          ? await graphqlRequest<{
              requestEmailVerificationCode: { resendAfterSeconds: number }
            }>(
              `mutation RequestEmailVerificationCode($input: RequestEmailVerificationCodeInput!) {
            requestEmailVerificationCode(input: $input) { resendAfterSeconds }
          }`,
              { input: { email } },
            )
          : await graphqlRequest<{
              requestLoginCode: { resendAfterSeconds: number }
            }>(
              `mutation RequestLoginCode($input: RequestLoginCodeInput!) {
            requestLoginCode(input: $input) { resendAfterSeconds }
          }`,
              { input: { email } },
            )
      const cooldown =
        "requestEmailVerificationCode" in data
          ? data.requestEmailVerificationCode.resendAfterSeconds
          : data.requestLoginCode.resendAfterSeconds
      setRemainingSeconds(cooldown)
      setCode("")
    } catch (resendError) {
      setError(
        resendError instanceof Error
          ? resendError.message
          : "Kod gönderilemedi",
      )
    } finally {
      setIsResending(false)
    }
  }

  return (
    <FormShell
      title={
        mode === "email-verification"
          ? "E-postanızı doğrulayın"
          : "Giriş kodunu girin"
      }
      description={`${email || "E-posta"} adresine 6 haneli kod gönderildi.`}
      footer={
        remainingSeconds > 0 ? (
          <span>Kodu tekrar göndermek için {remainingSeconds} saniye</span>
        ) : (
          <button
            type="button"
            onClick={handleResend}
            disabled={isResending}
            className="text-primary underline underline-offset-4"
          >
            {isResending ? "Gönderiliyor…" : "Kodu tekrar gönder"}
          </button>
        )
      }
    >
      <form onSubmit={handleSubmit} className="space-y-5" noValidate>
        <div className="flex justify-center">
          <InputOTP
            maxLength={codeLength}
            value={code}
            onChange={(value) => {
              setCode(value)
              if (error) setError(null)
            }}
            autoFocus
          >
            <InputOTPGroup>
              {Array.from({ length: codeLength }, (_, order) => (
                <InputOTPSlot
                  key={order}
                  index={order}
                  className="size-11 text-base"
                />
              ))}
            </InputOTPGroup>
          </InputOTP>
        </div>
        {error ? (
          <p className="text-center text-sm text-destructive" role="alert">
            {error}
          </p>
        ) : null}
        <p className="prova-meta text-center normal-case">
          {mode === "email-verification"
            ? "Kod 5 dakika geçerlidir ve yalnızca e-posta doğrulaması için kullanılabilir."
            : "Giriş kodu 10 dakika geçerlidir."}
        </p>
        <div className="pt-1">
          <Button
            type="submit"
            className="w-full"
            size="lg"
            disabled={isSubmitting}
          >
            {isSubmitting ? "Doğrulanıyor…" : "Doğrula"}
          </Button>
        </div>
      </form>
    </FormShell>
  )
}
