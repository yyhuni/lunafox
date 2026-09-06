import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "lib/api-client.ts"), "utf8")

describe("api-client contract", () => {
  it("uses shared auth storage keys instead of file-local token literals", () => {
    expect(source).toContain("from 'axios'")
    expect(source).toContain("from '@/lib/auth-storage-keys.mjs'")
    expect(source).toContain("from '@/lib/auth-runtime.mjs'")
    expect(source).not.toContain("const ACCESS_TOKEN_KEY = 'accessToken'")
    expect(source).not.toContain("const REFRESH_TOKEN_KEY = 'refreshToken'")
  })

  it("redirects protected 401 flows through the shared localized login redirect helper", () => {
    expect(source).toContain("buildLoginRedirectPath")
    expect(source).toContain("buildLoginRedirectPathForLocation")
    expect(source).not.toContain("window.location.href = getLoginPath()")
  })

  it("targets the backend v1 API and renews sessions through the canonical route", () => {
    expect(source).toContain("baseURL: '/v1'")
    expect(source).toContain("axios.post('/v1/sessions:renew'")
    expect(source).not.toContain("baseURL: '/api'")
    expect(source).not.toContain("/api/auth/refresh")
  })

  it("keeps one shared freshness owner and excludes session endpoints", () => {
    expect(source).toContain("TOKEN_REFRESH_LEAD_MS = 60_000")
    expect(source).toContain("ensurePrimaryTokenFresh")
    expect(source).toContain("isSessionAuthRequest")
    expect(source).toContain("activePrimaryTokenRenewal")
  })

  it("waits for mock network interception in the transport layer instead of blocking the React tree", () => {
    expect(source).toContain("from '@/mock/config'")
    expect(source).toContain('import("@/mock/browser")')
    expect(source).toContain("await ensureMockInterceptionReady()")
  })
})
