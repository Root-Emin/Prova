import { NextResponse } from "next/server"

export const runtime = "nodejs"

export async function POST() {
  const response = NextResponse.json({ ok: true })
  response.cookies.set({
    name: "prova_admin_session",
    value: "",
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: 0,
  })
  return response
}

