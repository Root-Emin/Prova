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
import { verificationInfo } from "./mock"

export default function VerifyPage() {
  const router = useRouter()
  const [code, setCode] = React.useState("")
  const [error, setError] = React.useState<string | null>(null)
  const [remainingSeconds, setRemainingSeconds] = React.useState(
    verificationInfo.resendSeconds
  )

  React.useEffect(() => {
    if (remainingSeconds <= 0) return
    const ticker = window.setInterval(() => {
      setRemainingSeconds((prev) => prev - 1)
    }, 1000)
    return () => window.clearInterval(ticker)
  }, [remainingSeconds])

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (code.length !== verificationInfo.codeLength) {
      setError("6 haneli kodu girin")
      return
    }
    setError(null)
    router.push("/verify-result")
  }

  return (
    <FormShell
      title="E-postanızı doğrulayın"
      description={`${verificationInfo.email} adresine 6 haneli kod gönderildi.`}
      footer={
        remainingSeconds > 0 ? (
          <span>Kodu tekrar göndermek için {remainingSeconds} saniye</span>
        ) : (
          <button
            type="button"
            onClick={() =>
              setRemainingSeconds(verificationInfo.resendSeconds)
            }
            className="text-primary underline underline-offset-4"
          >
            Kodu tekrar gönder
          </button>
        )
      }
    >
      <form onSubmit={handleSubmit} className="space-y-5" noValidate>
        <div className="flex justify-center">
          <InputOTP
            maxLength={verificationInfo.codeLength}
            value={code}
            onChange={(value) => {
              setCode(value)
              if (error) setError(null)
            }}
            autoFocus
          >
            <InputOTPGroup>
              {Array.from({ length: verificationInfo.codeLength }, (_, order) => (
                <InputOTPSlot key={order} index={order} className="size-11 text-base" />
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
          Kod 10 dakika geçerlidir. E-postadaki bağlantıya tıklayarak da
          doğrulayabilirsiniz.
        </p>
        <div className="pt-1">
          <Button type="submit" className="w-full" size="lg">
            Doğrula
          </Button>
        </div>
      </form>
    </FormShell>
  )
}
