import { NextResponse } from "next/server"

import { readSession, sessionCookie } from "@/lib/server-auth"

export const runtime = "nodejs"

export async function GET(request: Request) {
  const cookie = request.headers
    .get("cookie")
    ?.split(";")
    .map((part) => part.trim())
    .find((part) => part.startsWith(`${sessionCookie}=`))
    ?.slice(`${sessionCookie}=`.length)

  const session = readSession(cookie)
  if (!session) {
    return NextResponse.json({ authenticated: false }, { status: 401 })
  }

  return NextResponse.json({ authenticated: true, ...session })
}
