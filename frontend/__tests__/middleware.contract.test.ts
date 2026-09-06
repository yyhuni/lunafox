import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "middleware.ts"), "utf8")

describe("middleware contract", () => {
  it("redirects the root app entry before the App Router shell renders", () => {
    expect(source).toContain('import type { NextRequest } from "next/server"')
    expect(source).toContain('import { NextResponse } from "next/server"')
    expect(source).toContain("export default function middleware")
    expect(source).toContain("request: NextRequest")
    expect(source).toContain('request.nextUrl.pathname === "/"')
    expect(source).toContain("request.nextUrl.clone()")
    expect(source).toContain('redirectUrl.pathname = "/overview/"')
    expect(source).toContain("NextResponse.redirect(redirectUrl)")
  })

  it("keeps middleware narrow so canonical routes are not rewritten back to locale-prefixed paths", () => {
    expect(source).toContain("NextResponse.next()")
    expect(source).not.toContain("createMiddleware(routing)")
    expect(source).not.toContain("redirectLegacyLocalePrefix")
  })
})
