import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "scripts/run-interaction-smoke.mjs"), "utf8")
const devServerSource = readFileSync(path.resolve(process.cwd(), "scripts/smoke-dev-server.mjs"), "utf8")

describe("run-interaction-smoke contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from \"node:fs/promises\"")
  })

  it("uses localhost as default base url", () => {
    expect(source).toContain("http://127.0.0.1:3000")
  })

  it("does not fail routes only because no safe interaction was clicked", () => {
    expect(source).not.toContain("no-safe-interactions")
  })

  it("supports strict mode switch for exit behavior", () => {
    expect(source).toContain("const strictMode = process.env.INTERACTION_STRICT === \"1\"")
    expect(source).toContain("if (failed.length > 0 && strictMode)")
  })

  it("bounds each route so one stuck interaction cannot hang the smoke run", () => {
    expect(source).toContain("INTERACTION_ROUTE_TIMEOUT_MS")
    expect(source).toContain("interaction-route-timeout")
    expect(source).toContain("routeTimeoutMs")
  })

  it("supports desktop and mobile viewport smoke profiles", () => {
    expect(source).toContain("const viewportPresets = {")
    expect(source).toContain("INTERACTION_VIEWPORT")
    expect(source).toContain("SMOKE_VIEWPORT")
    expect(source).toContain("INTERACTION_VIEWPORT_WIDTH")
    expect(source).toContain("const viewportConfig = resolveViewportConfig()")
    expect(source).toContain("width: viewportConfig.width")
    expect(source).toContain("isMobile: viewportConfig.isMobile")
    expect(source).toContain("viewport=${viewportConfig.name}")
    expect(source).toContain("viewport: viewportConfig")
  })

  it("skips redirect-only routes that are covered by route smoke", () => {
    expect(source).toContain("redirectOnlyInteractionRoutePatterns")
    expect(source).toContain("\"/\"")
    expect(source).toContain("\"/login\"")
    expect(source).not.toContain("\"/scan/history/[id]\"")
    expect(source).not.toContain("\"/targets/[id]\"")
    expect(source).not.toContain("\"/targets/[id]/details\"")
    expect(source).not.toContain("\"/tools/fingerprints\"")
    expect(source).toContain("!redirectOnlyInteractionRoutePatterns.has(route.routePattern)")
  })

  it("defaults to deterministic serial concurrency", () => {
    expect(source).toContain("const concurrency = Number.parseInt(process.env.INTERACTION_CONCURRENCY || \"1\", 10)")
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

  it("uses shared smoke auth bootstrap helpers", () => {
    expect(source).toContain("from \"../lib/auth-runtime.mjs\"")
    expect(source).toContain("primeSmokeAuthSession")
    expect(source).toContain("primeSmokeLocaleCookie")
  })

  it("guards against concurrent smoke runs with a lock file", () => {
    expect(source).toContain(".interaction-smoke.lock")
    expect(source).toContain("fs.open(runLockPath, \"wx\")")
  })

  it("writes the interaction report atomically", () => {
    expect(source).toContain("tempReportPath")
    expect(source).toContain("fs.rename(tempReportPath, reportPath)")
  })
})
