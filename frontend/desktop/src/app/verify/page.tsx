"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import {
  InputOTP,
  InputOTPGroup,
  InputOTPSlot,
} from "@/components/ui/input-otp";
import { FormShell } from "@/components/prova/form-shell";
import {
  clearPendingLogin,
  getPendingLogin,
  setPendingLogin,
} from "@/lib/auth-flow";
import { authErrorMessage } from "@/lib/auth-errors";

const codeLength = 6;

export default function VerifyPage() {
  const router = useRouter();
  const [email, setEmail] = React.useState("");
  const [code, setCode] = React.useState("");
  const [remainingSeconds, setRemainingSeconds] = React.useState(0);
  const [error, setError] = React.useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  const [isResending, setIsResending] = React.useState(false);

  React.useEffect(() => {
    const pending = getPendingLogin();
    if (!pending) {
      setError("E-posta bilgisi bulunamadı. Lütfen işlemi baştan başlatın.");
      return;
    }
    setEmail(pending.email);
    setRemainingSeconds(pending.resendAfterSeconds);
  }, []);

  React.useEffect(() => {
    if (remainingSeconds <= 0) return;
    const ticker = window.setInterval(() => {
      setRemainingSeconds((prev) => prev - 1);
    }, 1000);
    return () => window.clearInterval(ticker);
  }, [remainingSeconds]);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!email) {
      setError("E-posta bilgisi bulunamadı. Lütfen işlemi baştan başlatın.");
      return;
    }
    if (!/^\d{6}$/.test(code)) {
      setError("6 haneli kodu girin");
      return;
    }
    const api = window.prova;
    if (!api) {
      setError("Masaüstü bağlantısı hazır değil; uygulamayı yeniden açın");
      return;
    }

    setError(null);
    setIsSubmitting(true);
    try {
      await api.auth.verifyLoginCode(email, code);
      clearPendingLogin();
      router.push("/");
    } catch (verifyError) {
      setError(
        authErrorMessage(
          verifyError,
          "Kod doğrulanamadı. Lütfen tekrar deneyin.",
        ),
      );
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleResend() {
    if (!email || isResending || remainingSeconds > 0) return;
    const api = window.prova;
    if (!api) {
      setError("Masaüstü bağlantısı hazır değil; uygulamayı yeniden açın");
      return;
    }

    setError(null);
    setIsResending(true);
    try {
      const response = await api.auth.requestLoginCode(email);
      setPendingLogin(email, response);
      setRemainingSeconds(response.resendAfterSeconds);
      setCode("");
    } catch (resendError) {
      setError(
        authErrorMessage(
          resendError,
          "Kod gönderilemedi. Lütfen tekrar deneyin.",
        ),
      );
    } finally {
      setIsResending(false);
    }
  }

  return (
    <FormShell
      title="Kodu girin"
      description={
        email
          ? `${email} adresine 6 haneli kod gönderildi.`
          : "E-posta adresinize 6 haneli kod gönderildi."
      }
      footer={
        <>
          {remainingSeconds > 0 ? (
            <span>Kodu tekrar göndermek için {remainingSeconds} saniye</span>
          ) : (
            <button
              type="button"
              onClick={handleResend}
              disabled={isResending || !email}
              className="text-primary underline underline-offset-4 disabled:opacity-50"
            >
              {isResending ? "Gönderiliyor…" : "Kodu tekrar gönder"}
            </button>
          )}
          <div className="mt-2">
            <Link
              href="/login"
              onClick={clearPendingLogin}
              className="text-primary underline underline-offset-4"
            >
              E-posta adresini değiştir
            </Link>
          </div>
        </>
      }
    >
      <form onSubmit={handleSubmit} className="space-y-5" noValidate>
        <div className="flex justify-center">
          <InputOTP
            maxLength={codeLength}
            value={code}
            onChange={(value) => {
              setCode(value);
              if (error) setError(null);
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
          Kod 10 dakika geçerlidir ve en fazla 5 deneme yapılabilir. Doğrulama
          sonunda bu cihaz hesabınıza otomatik olarak bağlanır.
        </p>
        <Button
          type="submit"
          className="w-full"
          size="lg"
          disabled={isSubmitting || code.length !== codeLength}
        >
          {isSubmitting ? "Doğrulanıyor…" : "Doğrula ve giriş yap"}
        </Button>
      </form>
    </FormShell>
  );
}
