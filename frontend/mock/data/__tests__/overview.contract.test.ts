import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "mock/data/overview.ts"), "utf8")

describe("overview contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function getMockStatisticsHistory")
    expect(source).toContain("from '@/types/overview.types'")
  })

  it("exports server runtime metrics for overview mock mode", () => {
    expect(source).toContain("mockServerRuntimeMetrics")
    expect(source).toContain("ServerRuntimeMetrics")
    expect(source).toContain('scope: "runtime"')
    expect(source).toContain("sampleIntervalSeconds: 2")
    expect(source).toContain("retentionSeconds: 600")
  })
})
