import { NextResponse } from "next/server"
import type { NextRequest } from "next/server"

export function proxy(request: NextRequest) {
  if (request.cookies.has("prova_admin_session")) {
    return NextResponse.next()
  }

  const loginURL = new URL("/login", request.url)
  loginURL.searchParams.set("next", request.nextUrl.pathname)
  return NextResponse.redirect(loginURL)
}

export const config = {
  matcher: [
    "/",
    "/assignments/:path*",
    "/audit-log/:path*",
    "/certificates/:path*",
    "/devices/:path*",
    "/personas/:path*",
    "/sessions/:path*",
    "/users/:path*",
  ],
}

