"use client";

import * as React from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { FormShell } from "@/components/prova/form-shell";
import { setPendingLogin } from "@/lib/auth-flow";
import { authErrorMessage } from "@/lib/auth-errors";

function emailLooksValid(value: string) {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value.trim());
}

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = React.useState("");
  const [error, setError] = React.useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  const validEmail = emailLooksValid(email);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalizedEmail = email.trim().toLowerCase();
    if (!emailLooksValid(normalizedEmail)) {
      setError("Geçerli bir e-posta girin");
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
      const response = await api.auth.requestLoginCode(normalizedEmail);
      setPendingLogin(normalizedEmail, response);
      router.push("/verify");
    } catch (requestError) {
      setError(
        authErrorMessage(
          requestError,
          "Kod gönderilemedi. Lütfen tekrar deneyin.",
        ),
      );
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <FormShell
      title="Giriş yap"
      description="Hesabınıza giriş yapmak için e-posta adresinizi yazın."
    >
      <form onSubmit={handleSubmit} className="space-y-5" noValidate>
        <div className="space-y-2">
          <Label htmlFor="login-email">E-posta adresi</Label>
          <Input
            id="login-email"
            type="email"
            inputMode="email"
            autoComplete="email"
            autoFocus
            value={email}
            onChange={(event) => {
              setEmail(event.target.value);
              if (error) setError(null);
            }}
            placeholder="ad.soyad@kurum.tr"
            aria-invalid={error ? true : undefined}
            aria-describedby={error ? "login-email-error" : undefined}
          />
          {error ? (
            <p
              id="login-email-error"
              className="text-sm text-destructive"
              role="alert"
            >
              {error}
            </p>
          ) : null}
        </div>

        <Button
          type="submit"
          className="w-full"
          size="lg"
          disabled={!validEmail || isSubmitting}
        >
          {isSubmitting ? "Gönderiliyor…" : "Kod gönder"}
        </Button>
      </form>
    </FormShell>
  );
}
