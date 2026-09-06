import type { NextRequest } from "next/server"
import { NextResponse } from "next/server"

export default function middleware(request: NextRequest) {
  if (request.nextUrl.pathname === "/") {
    const redirectUrl = request.nextUrl.clone()
    redirectUrl.pathname = "/overview/"
    return NextResponse.redirect(redirectUrl)
  }

  return NextResponse.next()
}

export const config = {
  // Match all paths, excluding API, static files, Next.js internal paths, etc.
  matcher: [
    // Match all paths
    "/((?!api|_next|_vercel|.*\\..*).*)",
  ],
}
