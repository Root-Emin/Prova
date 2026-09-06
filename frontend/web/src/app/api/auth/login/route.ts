import { NextResponse } from "next/server"
import {
  makeSession,
  sessionCookie,
  sessionLifetimeSeconds,
} from "@/lib/server-auth"

export const runtime = "nodejs"

function getCredential(name: string, fallback: string): string {
  return process.env[name] ?? fallback
}

export async function POST(request: Request) {
  let body: { email?: unknown; password?: unknown }
  try {
    body = (await request.json()) as { email?: unknown; password?: unknown }
  } catch {
    return NextResponse.json({ message: "Geçersiz istek." }, { status: 400 })
  }

  const email = typeof body.email === "string" ? body.email.trim().toLowerCase() : ""
  const password = typeof body.password === "string" ? body.password : ""
  const adminEmail = getCredential("ADMIN_EMAIL", "yonetici@prova.local").toLowerCase()
  const adminPassword = getCredential("ADMIN_PASSWORD", "SecurePass123!")
  const employeeEmail = getCredential("EMPLOYEE_EMAIL", "calisan@prova.local").toLowerCase()

  if (email === employeeEmail) {
    return NextResponse.json(
      { message: "Çalışan hesabı web panelinden değil, Prova Desktop uygulamasından kullanılır." },
      { status: 403 },
    )
  }

  if (email !== adminEmail || password !== adminPassword) {
    return NextResponse.json(
      { message: "E-posta veya şifre hatalı." },
      { status: 401 },
    )
  }

  const response = NextResponse.json({
    email: adminEmail,
    name: "Demo Yönetici",
    role: "admin",
  })
  response.cookies.set({
    name: sessionCookie,
    value: makeSession(adminEmail),
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: sessionLifetimeSeconds,
  })
  return response
}
