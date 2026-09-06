import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-auth.ts"), "utf8")

describe("use-auth contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useAuth")
    expect(source).toContain("from '@tanstack/react-query'")
  })

  it("keeps auth token access behind the auth service boundary", () => {
    expect(source).toContain("hasAuthTokens")
    expect(source).toContain("from '@/services/auth.service'")
    expect(source).not.toContain("from '@/lib/api-client'")
    expect(source).not.toContain("tokenManager")
  })

  it("uses shared auth runtime for logout redirect", () => {
    expect(source).toContain("from '@/lib/auth-runtime.mjs'")
    expect(source).toContain("pushWithRouteProgress(router, buildLoginPath())")
    expect(source).not.toContain("buildLoginPath(locale)")
    expect(source).not.toContain("router.push(`/${locale}/login/`)")
  })
})
