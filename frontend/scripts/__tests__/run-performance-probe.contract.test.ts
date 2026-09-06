import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "scripts/run-performance-probe.mjs"), "utf8")

describe("run-performance-probe contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from \"node:fs/promises\"")
  })

  it("collects web vitals, long task, and resource payload evidence", () => {
    expect(source).toContain("largest-contentful-paint")
    expect(source).toContain("layout-shift")
    expect(source).toContain("longtask")
    expect(source).toContain("performance.getEntriesByType(\"resource\")")
    expect(source).toContain("resourceSummary")
    expect(source).toContain("largestContentfulPaintMs")
    expect(source).toContain("cumulativeLayoutShift")
    expect(source).toContain("longTaskTotalMs")
  })

  it("applies named browser profiles instead of only recording the profile label", () => {
    expect(source).toContain("const browserProfiles = {")
    expect(source).toContain('"desktop-chromium"')
    expect(source).toContain('"mobile-chromium"')
    expect(source).toContain("viewport: { width: 390, height: 844 }")
    expect(source).toContain("await browser.newContext(selectedBrowserProfile)")
  })

  it("uses the shared route-pattern materializer for dynamic smoke targets", () => {
    expect(source).toContain('from "./route-pattern.mjs"')
    expect(source).toContain("instantiateRoutePattern(route.routePattern)")
    expect(source).not.toContain("function normalizeRoute")
  })
})
