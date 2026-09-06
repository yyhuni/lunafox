import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "scripts/run-route-smoke.mjs"), "utf8")
const devServerSource = readFileSync(path.resolve(process.cwd(), "scripts/smoke-dev-server.mjs"), "utf8")
const packageJson = JSON.parse(
  readFileSync(path.resolve(process.cwd(), "package.json"), "utf8")
) as { scripts: Record<string, string> }

describe("run-route-smoke contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from \"node:fs/promises\"")
  })

  it("uses localhost as default base url", () => {
    expect(source).toContain("http://127.0.0.1:3000")
  })

  it("supports including passed routes and requiring non-empty target set", () => {
    expect(source).toContain("const includePassed = process.env.SMOKE_INCLUDE_PASSED === \"1\"")
    expect(source).toContain("const requireTargets = process.env.SMOKE_REQUIRE_TARGETS === \"1\"")
    expect(source).toContain("no targets to run (SMOKE_REQUIRE_TARGETS=1)")
    expect(packageJson.scripts["test:e2e:routes"]).toBe(
      "SMOKE_INCLUDE_PASSED=1 SMOKE_REQUIRE_TARGETS=1 SMOKE_CONCURRENCY=1 node scripts/run-route-smoke.mjs"
    )
  })

  it("supports desktop and mobile viewport smoke profiles", () => {
    expect(source).toContain("const viewportPresets = {")
    expect(source).toContain("SMOKE_VIEWPORT")
    expect(source).toContain("SMOKE_VIEWPORT_WIDTH")
    expect(source).toContain("SMOKE_VIEWPORT_HEIGHT")
    expect(source).toContain("const viewportConfig = resolveViewportConfig()")
    expect(source).toContain("width: viewportConfig.width")
    expect(source).toContain("isMobile: viewportConfig.isMobile")
    expect(source).toContain("viewport=${viewportConfig.name}")
    expect(source).toContain("viewport: viewportConfig")
  })

  it("auto-starts dev server in skip-auth mode for localhost smoke runs", () => {
    expect(source).toContain("from \"./smoke-dev-server.mjs\"")
    expect(source).toContain("acquireSmokeDevServerLease")
    expect(source).toContain("releaseDevServerLease")
    expect(devServerSource).toContain("NEXT_PUBLIC_SKIP_AUTH: \"true\"")
    expect(devServerSource).toContain('NEXT_PUBLIC_USE_MOCK: process.env.NEXT_PUBLIC_USE_MOCK ?? "true"')
    expect(devServerSource).toContain('NEXT_PUBLIC_MOCK_SCENARIO: process.env.NEXT_PUBLIC_MOCK_SCENARIO ?? "happy"')
    expect(devServerSource).toContain("smoke-dev-server-state.json")
    expect(devServerSource).toContain("leases")
    expect(source).not.toContain("function startDevServer")
  })

  it("uses the shared route-pattern materializer for dynamic smoke targets", () => {
    expect(source).toContain('from "./route-pattern.mjs"')
    expect(source).toContain("instantiateRoutePattern(route.routePattern)")
    expect(source).not.toContain("function normalizeRoute")
  })

  it("uses stable auth bootstrap and quick scan selectors", () => {
    expect(source).toContain("from \"../lib/auth-runtime.mjs\"")
    expect(source).toContain("primeSmokeAuthSession")
    expect(source).toContain("primeSmokeLocaleCookie")
    expect(source).toContain("data-slot='quick-scan-trigger'")
    expect(source).toContain("data-slot='quick-scan-drawer'")
    expect(source).toContain("quick-scan-drawer-not-opened")
    expect(source).toContain("missing-quick-scan-trigger")
    expect(source).toContain("unexpected-auth-redirect")
  })
})
