import { describe, expect, it } from "vitest"

import {
  SMOKE_AUTH_SESSION_ENABLED,
  SMOKE_AUTH_SESSION_KEY,
  buildLoginRedirectPath,
  buildLoginRedirectPathForLocation,
  buildReturnTo,
  createSmokeLocaleCookie,
  buildLoginPath,
  isPublicPathname,
  primeSmokeAuthSession,
  SMOKE_LOCALE_COOKIE_KEY,
  resolveSafeReturnTo,
} from "@/lib/auth-runtime.mjs"
import {
  AUTH_PRIMARY_TOKEN_KEY,
  AUTH_RENEWAL_TOKEN_KEY,
} from "@/lib/auth-storage-keys.mjs"

describe("auth-runtime", () => {
  it("recognizes canonical public auth routes only", () => {
    expect(isPublicPathname("/login")).toBe(true)
    expect(isPublicPathname("/login/reset")).toBe(true)
    expect(isPublicPathname("/legacy/login")).toBe(false)
    expect(isPublicPathname("/overview")).toBe(false)
  })

  it("builds canonical locale-neutral login paths", () => {
    expect(buildLoginPath()).toBe("/login/")
  })

  it("builds canonical returnTo login redirects for protected routes", () => {
    expect(buildReturnTo("/targets/42/overview/", "tab=summary", "section-1")).toBe(
      "/targets/42/overview/?tab=summary#section-1"
    )

    expect(buildLoginRedirectPath("/targets/42/overview/?tab=summary")).toBe(
      "/login/?returnTo=%2Ftargets%2F42%2Foverview%2F%3Ftab%3Dsummary"
    )
  })

  it("does not wrap public auth routes into recursive returnTo redirects", () => {
    expect(buildLoginRedirectPathForLocation("/login/", "returnTo=%2Ftargets%2F")).toBe(
      "/login/"
    )
    expect(buildLoginRedirectPathForLocation("/login/reset", "step=verify")).toBe(
      "/login/"
    )
    expect(buildLoginRedirectPathForLocation("/targets/", "tab=summary")).toBe(
      "/login/?returnTo=%2Ftargets%2F%3Ftab%3Dsummary"
    )
  })

  it("normalizes the root app entry to overview when building auth returnTo", () => {
    expect(buildLoginRedirectPathForLocation("/")).toBe(
      "/login/?returnTo=%2Foverview%2F"
    )
  })

  it("only accepts in-app canonical returnTo destinations", () => {
    expect(resolveSafeReturnTo("/targets/42/overview/?tab=summary")).toBe(
      "/targets/42/overview/?tab=summary"
    )
    expect(resolveSafeReturnTo("https://evil.example/phish")).toBeNull()
    expect(resolveSafeReturnTo("//evil.example/phish")).toBeNull()
    expect(resolveSafeReturnTo("javascript:alert(1)")).toBeNull()
    expect(resolveSafeReturnTo("/")).toBe("/overview/")
  })

  it("creates locale cookie payloads for canonical smoke routes", () => {
    expect(createSmokeLocaleCookie("http://127.0.0.1:3000", "en")).toEqual({
      name: SMOKE_LOCALE_COOKIE_KEY,
      value: "en",
      url: "http://127.0.0.1:3000",
      sameSite: "Lax",
      secure: false,
    })
  })

  it("primes smoke auth session with stable storage keys", () => {
    const storage = new Map<string, string>()
    const storageLike = {
      getItem(key: string) {
        return storage.get(key) ?? null
      },
      setItem(key: string, value: string) {
        storage.set(key, value)
      },
      removeItem(key: string) {
        storage.delete(key)
      },
    }

    primeSmokeAuthSession(storageLike)

    expect(storage.get(AUTH_PRIMARY_TOKEN_KEY)).toBe("mock-access-token")
    expect(storage.get(AUTH_RENEWAL_TOKEN_KEY)).toBe("mock-refresh-token")
    expect(storage.get(SMOKE_AUTH_SESSION_KEY)).toBe(SMOKE_AUTH_SESSION_ENABLED)
  })
})
