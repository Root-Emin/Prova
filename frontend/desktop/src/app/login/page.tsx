"use client";

import * as React from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { FormShell } from "@/components/prova/form-shell";
import { authErrorMessage } from "@/lib/auth-errors";

function emailLooksValid(value: string) {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value.trim());
}

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = React.useState("");
  const [password, setPassword] = React.useState("");
  const [error, setError] = React.useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  const validEmail = emailLooksValid(email);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalizedEmail = email.trim().toLowerCase();
    if (!emailLooksValid(normalizedEmail)) {
      setError("Geçerli bir kullanıcı adı/e-posta girin");
      return;
    }
    if (password.length < 8) {
      setError("Şifre en az 8 karakter olmalı");
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
      await api.auth.login(normalizedEmail, password);
      router.push("/");
    } catch (requestError) {
      setError(
        authErrorMessage(
          requestError,
          "Giriş yapılamadı. Lütfen bilgilerinizi kontrol edin.",
        ),
      );
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <FormShell
      title="Giriş yap"
      description="Test hesabınızın kullanıcı adı ve şifresiyle giriş yapın."
    >
      <form onSubmit={handleSubmit} className="space-y-5" noValidate>
        <div className="space-y-2">
          <Label htmlFor="login-email">Kullanıcı adı / e-posta</Label>
          <Input
            id="login-email"
            type="email"
            inputMode="email"
            autoComplete="username"
            autoFocus
            value={email}
            onChange={(event) => {
              setEmail(event.target.value);
              if (error) setError(null);
            }}
            placeholder="yonetici@prova.local"
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

        <div className="space-y-2">
          <Label htmlFor="login-password">Şifre</Label>
          <Input
            id="login-password"
            type="password"
            minLength={8}
            autoComplete="current-password"
            value={password}
            onChange={(event) => {
              setPassword(event.target.value);
              if (error) setError(null);
            }}
            placeholder="SecurePass123!"
          />
        </div>

        <Button
          type="submit"
          className="w-full"
          size="lg"
          disabled={!validEmail || isSubmitting}
        >
          {isSubmitting ? "Giriş yapılıyor…" : "Giriş yap"}
        </Button>
      </form>
    </FormShell>
  );
}
