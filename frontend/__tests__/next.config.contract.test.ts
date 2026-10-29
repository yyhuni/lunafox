import { afterEach, describe, expect, it, vi } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

vi.mock("next-intl/plugin", () => ({ default: () => (config: unknown) => config }))

const source = readFileSync(path.resolve(process.cwd(), "next.config.ts"), "utf8")
const vercelConfig = JSON.parse(readFileSync(path.resolve(process.cwd(), "vercel.json"), "utf8")) as {
  env?: Record<string, string>
}

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

  it("routes Vercel real-mode v1 requests through its configured backend", () => {
    expect(source).toContain("const useMock = process.env.NEXT_PUBLIC_USE_MOCK === 'true'")
    expect(source).toContain("getRequiredVercelBackendOrigin")
    expect(source).toContain("process.env.NODE_ENV === \"production\"")
    expect(source).toContain("destination: `${vercelBackendOrigin}/v1/:path*`")
    expect(source).toContain("if (useMock)")
  })

  it("keeps production Vercel builds on the real API and auth boundary", () => {
    expect(vercelConfig.env?.NEXT_PUBLIC_USE_MOCK).toBeUndefined()
    expect(vercelConfig.env?.NEXT_PUBLIC_SKIP_AUTH).toBeUndefined()
    expect(vercelConfig.env?.DEMO_SITE).toBeUndefined()
    expect(source).toContain("assertProductionRuntimeFlags")
    expect(source).toContain("const skipAuth = process.env.NEXT_PUBLIC_SKIP_AUTH === 'true'")
    expect(source).toContain("const demoSite = process.env.DEMO_SITE === 'true'")
    expect(source).toContain("assertVercelDemoHasNoBackend")
    expect(source).toContain("&& !demoSite")
    expect(source).not.toContain("NEXT_PUBLIC_BACKEND_URL")
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

describe("same-origin backend rewrites", () => {
  afterEach(() => {
    vi.unstubAllEnvs()
    vi.resetModules()
  })

  async function rewrites({ mock = false, vercel = false } = {}) {
    vi.stubEnv("NODE_ENV", vercel ? "production" : "development")
    vi.stubEnv("VERCEL", vercel ? "1" : "0")
    vi.stubEnv("DEMO_SITE", "false")
    vi.stubEnv("NEXT_PUBLIC_USE_MOCK", String(mock))
    vi.stubEnv("NEXT_PUBLIC_SKIP_AUTH", "false")
    vi.stubEnv("NEXT_PUBLIC_BACKEND_URL", "")
    vi.stubEnv("BACKEND_URL", "https://backend.example.com")
    vi.stubEnv("API_HOST", "server")
    const { default: config } = await import("../next.config")
    if (!config.rewrites) throw new Error("Backend rewrites are required")
    return config.rewrites()
  }

  it("forwards MCP and key API requests to the same local backend", async () => {
    expect(await rewrites()).toEqual([
      { source: "/v1/:path*", destination: "http://server:8080/v1/:path*" },
      { source: "/mcp", destination: "http://server:8080/mcp" },
    ])
  })

  it("forwards MCP and key API requests to the configured Vercel backend", async () => {
    expect(await rewrites({ vercel: true })).toEqual([
      { source: "/v1/:path*", destination: "https://backend.example.com/v1/:path*" },
      { source: "/mcp", destination: "https://backend.example.com/mcp" },
    ])
  })

  it("keeps mock mode from forwarding any request to a real backend", async () => {
    expect(await rewrites({ mock: true })).toEqual([])
  })
})
