import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/target/target-overview-sections.tsx"), "utf8")

describe("target-overview-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function TargetOverviewLoadingState")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("routes severity and status affordances through shared helpers", () => {
    expect(source).toContain("getSeverityColor")
    expect(source).toContain("getStatusToneTextClass")
    expect(source).not.toContain("text-green-600")
    expect(source).not.toContain("text-green-500")
    expect(source).not.toContain("bg-red-500")
    expect(source).not.toContain("bg-orange-500")
    expect(source).not.toContain("bg-yellow-500")
    expect(source).not.toContain("bg-blue-500")
  })

  it("uses approved Button sizes for compact card actions", () => {
    expect(source).not.toContain('className="h-7 text-xs"')
  })

  it("keeps the overview skeleton aligned with the scan history section", () => {
    expect(source).toContain("ScanHistoryListLoadingState")
    expect(source).toContain('from "@/components/scan/history/scan-history-list-sections"')
    expect(source).toContain('owner="target-overview-scan-history"')
    expect(source).toContain('layer="section"')
    expect(source).toContain("hideToolbar")
    expect(source).toContain("hideTargetColumn")
    expect(source).not.toContain("ScanHistoryListSkeleton")
    expect(source).not.toContain("scan-history-list-skeleton")
  })

  it("uses the shared structural action placeholder for the initial scan CTA", () => {
    expect(source).toContain('from "@/components/shared/loading/action-skeleton"')
    expect(source).toContain('<ActionSkeleton widthClassName="w-36" />')
    expect(source).not.toContain('Skeleton className="h-9 w-36')
  })

  it("derives the loading state from the same overview section shells as resolved content", () => {
    expect(source).toContain("function TargetOverviewTopBar")
    expect(source).toContain("function TargetOverviewSection")
    expect(source).toContain("function TargetAssetCardShell")
    expect(source.match(/<TargetOverviewSection/g)?.length ?? 0).toBeGreaterThanOrEqual(4)
    expect(source.match(/<TargetAssetCardShell/g)?.length ?? 0).toBeGreaterThanOrEqual(2)
  })

  it("avoids layout-shifting hover motion", () => {
    expect(source).not.toContain("group-hover:w-")
    expect(source).not.toContain("group-hover:translate-x")
  })
})
