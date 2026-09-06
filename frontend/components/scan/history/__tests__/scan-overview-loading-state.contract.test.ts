import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-overview-loading-state.tsx"), "utf8")
const overviewSource = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-overview.tsx"), "utf8")
const runtimeSource = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-runtime-detail-drawer.tsx"), "utf8")
const layoutSource = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-overview-layout.ts"), "utf8")

describe("scan-overview-loading-state contract", () => {
  it("owns the lightweight overview skeleton used by route fallbacks", () => {
    expect(source).toContain("export function ScanOverviewLoadingState")
    expect(source).toContain('from "@/components/ui/skeleton"')
    expect(source).toContain('from "@/components/shared/detail-drawer"')
    expect(source).toContain('from "@/lib/utils"')
    expect(source).not.toContain("useScanOverviewState")
    expect(source).not.toContain("useTaskProgressLogs")
    expect(source).not.toContain('from "@/components/scan/history/scan-runtime-detail-drawer"')
    expect(source).not.toContain('from "@/components/scan/history/scan-overview-sections"')
  })

  it("derives runtime detail tab loading controls from shared detail drawer tabs", () => {
    expect(source).toContain("DetailDrawerTabsList")
    expect(source).toContain("DetailDrawerTabsTrigger")
    expect(source).toContain('<DetailDrawerTabs value="logs" aria-hidden="true">')
    expect(source).toContain('<DetailDrawerTabsList variant="minimal" size="md" className="min-w-0 max-w-full overflow-x-auto">')
    expect(source).toContain('<DetailDrawerTabsTrigger variant="minimal" activeIndicator="fixed" value="logs" size="md" disabled>')
    expect(source).toContain('<DetailDrawerTabsTrigger variant="minimal" activeIndicator="fixed" value="config" size="md" disabled>')
    expect(source).not.toContain('Array.from({ length: 4 }).map((_, index) => (\n                <Skeleton key={index} className="h-8 w-20 rounded-md" />')
    expect(source).not.toContain('Skeleton className="h-9 w-24 rounded-md"')
  })

  it("keeps the overview loading shell aligned with the xl side-panel layout", () => {
    expect(layoutSource).toContain("SCAN_OVERVIEW_WORKBENCH_CLASS")
    expect(layoutSource).toContain("SCAN_OVERVIEW_PRIMARY_COLUMN_CLASS")
    expect(layoutSource).toContain("SCAN_OVERVIEW_SIDE_PANEL_CLASS")
    expect(source).toContain("SCAN_OVERVIEW_WORKBENCH_CLASS")
    expect(overviewSource).toContain("SCAN_OVERVIEW_WORKBENCH_CLASS")
    expect(source).toContain("SCAN_OVERVIEW_PRIMARY_COLUMN_CLASS")
    expect(overviewSource).toContain("SCAN_OVERVIEW_PRIMARY_COLUMN_CLASS")
    expect(source).toContain("SCAN_OVERVIEW_SIDE_PANEL_CLASS")
    expect(runtimeSource).toContain("SCAN_OVERVIEW_STICKY_SIDE_PANEL_CLASS")
    expect(source).toContain("sectionIndex === 0 || sectionIndex === 3 ? 3 : 4")
    expect(source).not.toContain("2xl:flex-row")
    expect(source).not.toContain("2xl:w-80")
  })

  it("keeps the overview skeleton aligned with the responsive summary rhythm", () => {
    expect(layoutSource).toContain("SCAN_OVERVIEW_SUMMARY_GRID_CLASS")
    expect(layoutSource).toContain("SCAN_OVERVIEW_SUMMARY_ITEM_CLASS")
    expect(layoutSource).toContain("sm:grid-cols-6")
    expect(source).toContain("SCAN_OVERVIEW_SUMMARY_GRID_CLASS")
    expect(runtimeSource).toContain("SCAN_OVERVIEW_SUMMARY_GRID_CLASS")
    expect(source).toContain("SCAN_OVERVIEW_SUMMARY_ITEM_CLASS")
    expect(runtimeSource).toContain("SCAN_OVERVIEW_SUMMARY_ITEM_CLASS")
    expect(source).toContain("Array.from({ length: 6 }).map")
    expect(source).toContain("Array.from({ length: 3 }).map")
    expect(source).not.toContain("2xl:grid-cols-7")
  })
})
