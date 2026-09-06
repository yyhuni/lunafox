import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/metrics/stat-metric-row.tsx"), "utf8")

describe("stat-metric-row contract", () => {
  it("owns the shared divided metric row used by scan history and vulnerability summaries", () => {
    expect(source).toContain("export function StatMetricRow")
    expect(source).toContain("export type StatMetricRowItem")
    expect(source).toContain("featuredKey?: string")
    expect(source).toContain("featured={key === featuredKey}")
    const itemType = source.match(/export type StatMetricRowItem = \{([\s\S]*?)\n\}/)?.[1] ?? ""
    expect(itemType).not.toContain("featured?: boolean")
    expect(source).toContain("divide-y divide-border border-y border-border")
    expect(source).toContain("MetricValueSkeleton")
    expect(source).not.toContain("ContentHandoff")
  })

  it("keeps loading metric values line-box compatible with resolved numeric values", () => {
    expect(source).toContain('featured ? "w-6" : "w-4"')
    expect(source).toContain('"loading-skeleton !absolute block left-0 top-1/2 -translate-y-1/2 rounded-md"')
    expect(source).toContain("text-transparent")
    expect(source).toContain("textRole.metricValueDisplay")
    expect(source).not.toContain('"text-4xl font-bold leading-none tabular-nums"')
    expect(source).not.toContain('"text-2xl font-semibold leading-none tabular-nums text-foreground"')
    expect(source).not.toContain("w-[1ch]")
    expect(source).not.toContain("min-w-[1ch]")
    expect(source).not.toContain("inline-block w-12 select-none")
  })
})
