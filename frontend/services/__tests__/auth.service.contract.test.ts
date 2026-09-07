import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "services/auth.service.ts"), "utf8")

describe("auth.service contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from '@/lib/api-client'")
  })

  it("owns auth token presence checks for hook callers", () => {
    expect(source).toContain("export function hasAuthTokens")
    expect(source).toContain("return tokenManager.hasTokens()")
  })

  it("exposes session freshness to hooks without leaking api-client imports", () => {
    expect(source).toContain("export function ensureFreshAuthToken")
    expect(source).toContain("export function authTokenExpirationMs")
    expect(source).toContain("export function authTokenRefreshDelayMs")
  })

  it("uses the backend canonical session and current-user routes", () => {
    expect(source).toContain("api.post<LoginResponse>('/sessions', data)")
    expect(source).toContain("api.get<{ id: number; username: string; email: string }>('/users/current')")
    expect(source).toContain("api.post<ChangePasswordResponse>('/users/me:changePassword', data)")
    expect(source).not.toContain("/auth/login")
    expect(source).not.toContain("/auth/me")
    expect(source).not.toContain("/users/me/password")
  })
})
