"use client"

import * as React from "react"
import { usePathname, useRouter } from "next/navigation"

export function AdminAuthGate({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const pathname = usePathname()
  const [status, setStatus] = React.useState<"checking" | "allowed">("checking")

  React.useEffect(() => {
    let cancelled = false

    fetch("/api/auth/session", { cache: "no-store" })
      .then((response) => {
        if (!response.ok) {
          const next = pathname === "/" ? "/" : pathname
          router.replace(`/login?next=${encodeURIComponent(next)}`)
          return
        }
        if (!cancelled) setStatus("allowed")
      })
      .catch(() => {
        router.replace(`/login?next=${encodeURIComponent(pathname)}`)
      })

    return () => {
      cancelled = true
    }
  }, [pathname, router])

  if (status !== "allowed") {
    return (
      <main className="prova-texture flex min-h-svh items-center justify-center bg-background px-6">
        <p className="prova-meta uppercase">Prova yönetim alanı hazırlanıyor</p>
      </main>
    )
  }

  return children
}

