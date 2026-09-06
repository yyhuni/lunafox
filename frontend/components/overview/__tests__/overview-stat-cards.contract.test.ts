import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/overview/overview-stat-cards.tsx"), "utf8")

describe("overview-stat-cards contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function OverviewStatCards")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses shared trend tone helpers for trend badges", () => {
    expect(source).toContain("getTrendToneBadgeClass")
    expect(source).not.toContain("text-[var(--success)]")
    expect(source).not.toContain("border-[var(--success)]")
    expect(source).not.toContain("text-[var(--error)]")
    expect(source).not.toContain("border-[var(--error)]")
    expect(source).not.toContain("getStatusToneBadgeClass")
  })

  it("uses the reference metric card composition", () => {
    expect(source).toContain("MetricSparkline")
    expect(source).toContain("textRole")
    expect(source).toContain("assetsFound")
    expect(source).toContain("vulnsFound")
    expect(source).toContain("monitoredTargets")
    expect(source).toContain("runningScans")
  })
})
