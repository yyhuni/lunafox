import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/auth/auth-guard.tsx"), "utf8")

describe("auth-guard contract", () => {
  it("uses shared auth runtime helpers instead of route-local redirect wiring", () => {
    expect(source).toContain("export function AuthGuard")
    expect(source).toContain("from \"@/lib/auth-runtime.mjs\"")
    expect(source).not.toContain("const PUBLIC_ROUTES")
    expect(source).not.toContain("router.push(loginPath)")
  })

  it("uses protected app shell warmup instead of login-form auth warmup geometry", () => {
    expect(source).toContain('import { AppShellWarmup } from "@/components/shared/loading/app-shell-warmup"')
    expect(source).toContain("const AUTH_GUARD_WARMUP_DELAY_MS = 180")
    expect(source).toContain('<AppShellWarmup owner="auth-guard-check"')
    expect(source).toContain('delayMs={AUTH_GUARD_WARMUP_DELAY_MS}')
    expect(source).toContain('<AppShellWarmup owner="auth-guard-redirect"')
    expect(source).not.toContain('import { AuthWarmupShell }')
    expect(source).not.toContain("<AuthWarmupShell")
  })
})
