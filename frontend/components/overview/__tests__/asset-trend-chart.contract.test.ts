import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/overview/asset-trend-chart.tsx"), "utf8")

describe("asset-trend-chart contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function AssetTrendChart")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses the shared Chinese compact axis formatter", () => {
    expect(source).toContain("formatAssetTrendAxisTick")
    expect(source).toContain("activeAxisMaxValue")
    expect(source).not.toContain("toFixed(1)}k")
  })

  it("uses trend semantics for displayed delta values", () => {
    const deltaSource = source.slice(
      source.indexOf("<IconTrendingUp"),
      source.indexOf("</span>", source.indexOf("<IconTrendingUp"))
    )

    expect(source).toContain("getTrendToneTextClass")
    expect(source).toContain('const ASSET_DELTA_TEXT_CLASS = getTrendToneTextClass("positive")')
    expect(source).toContain('className={cn("flex font-mono gap-1 items-center text-xs", ASSET_DELTA_TEXT_CLASS)}')
    expect(deltaSource).not.toContain("text-muted-foreground")
    expect(deltaSource).not.toContain("text-success")
  })
})
