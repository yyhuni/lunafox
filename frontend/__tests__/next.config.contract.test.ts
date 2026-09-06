import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "next.config.ts"), "utf8")

describe("next.config contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from \"next\"")
  })

  it("proxies the frontend v1 API prefix to the Go backend", () => {
    expect(source).toContain("source: '/v1/:path*'")
    expect(source).toContain("destination: `http://${apiHost}:8080/v1/:path*`")
    expect(source).not.toContain("source: '/api/:path*/'")
    expect(source).not.toContain("destination: `http://${apiHost}:8080/api/:path*/`")
  })

  it("does not register the backend proxy when mock mode owns v1 requests", () => {
    expect(source).toContain("const useMock = process.env.NEXT_PUBLIC_USE_MOCK === 'true'")
    expect(source).toContain("if (isVercel || useMock)")
  })

  it("allows local dev origins used by browser smoke and playwright", () => {
    expect(source).toContain("allowedDevOrigins")
    expect(source).toContain("'localhost'")
    expect(source).toContain("'127.0.0.1'")
    expect(source).toContain("'::1'")
  })

  it("does not retain bind-mount polling configuration", () => {
    expect(source).not.toContain("NEXT_DEV_POLLING")
    expect(source).not.toContain("watchOptions")
  })

  it("loads next-intl with a synchronous config plugin import", () => {
    expect(source).toContain('from "next-intl/plugin"')
    expect(source).not.toContain("await import(")
    expect(source).not.toContain("intlExtensionSegment")
  })
})
