import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/overview/asset-distribution-chart.tsx"), "utf8")

describe("asset-distribution-chart contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function AssetDistributionChart")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses shared chart owner helpers instead of local raw chart vars", () => {
    expect(source).toContain('from "@/lib/chart-config"')
    expect(source).toContain("getAssetDistributionChartColor")
    expect(source).not.toContain('subdomain: "var(--chart-1)"')
    expect(source).not.toContain('ip: "var(--chart-2)"')
    expect(source).not.toContain('endpoint: "var(--chart-3)"')
    expect(source).not.toContain('website: "var(--chart-4)"')
  })
})
