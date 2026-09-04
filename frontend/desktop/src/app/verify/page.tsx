"use client"

import * as React from "react"
import Link from "next/link"

import { Button } from "@/components/ui/button"
import {
  InputOTP,
  InputOTPGroup,
  InputOTPSlot,
} from "@/components/ui/input-otp"
import { FormShell } from "@/components/prova/form-shell"
import { useDeviceState } from "@/hooks/use-device-state"
import { verificationInfo } from "./mock"

export default function VerifyPage() {
  const device = useDeviceState()
  const [code, setCode] = React.useState("")
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

  const ok = code.length === verificationInfo.codeLength
  // Cihaz eşleştirme girişin tamamlandığı anda olur; kayıtsız cihaz önce
  // kayıt ekranına uğrar.
  const nextHref = device.registered ? "/" : "/device-enrollment"

  return (
    <FormShell
      title="Kodu girin"
      description={`${verificationInfo.email} adresine 6 haneli kod gönderildi.`}
      action={
        <Button
          nativeButton={ok ? false : undefined}
          className="w-full"
          size="lg"
          disabled={!ok}
          render={ok ? <Link href={nextHref} /> : undefined}
        >
          Doğrula ve giriş yap
        </Button>
      }
      footer={
        <>
          {remainingSeconds > 0 ? (
            <span>Kodu tekrar göndermek için {remainingSeconds} saniye</span>
          ) : (
            <button
              type="button"
              onClick={() => setRemainingSeconds(verificationInfo.resendSeconds)}
              className="text-primary underline underline-offset-4"
            >
              Kodu tekrar gönder
            </button>
          )}
          <div className="mt-2">
            <Link
              href="/login"
              className="text-primary underline underline-offset-4"
            >
              E-posta adresini değiştir
            </Link>
          </div>
        </>
      }
    >
      <div className="flex justify-center">
        <InputOTP
          maxLength={verificationInfo.codeLength}
          value={code}
          onChange={setCode}
          autoFocus
        >
          <InputOTPGroup>
            {Array.from({ length: verificationInfo.codeLength }, (_, order) => (
              <InputOTPSlot key={order} index={order} className="size-11 text-base" />
            ))}
          </InputOTPGroup>
        </InputOTP>
      </div>
      <p className="prova-meta text-center normal-case">
        Kod {verificationInfo.ttlMinutes} dakika geçerlidir, en fazla{" "}
        {verificationInfo.attemptLimit} deneme yapılabilir. Doğrulama sonunda bu
        cihaz hesabınıza bağlanır; yeni bir cihazda adresinize bilgilendirme
        postası gönderilir.
      </p>
    </FormShell>
  )
}
