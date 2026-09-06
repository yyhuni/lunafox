import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/overview/overview-scheduled-scans.tsx"), "utf8")

describe("overview-scheduled-scans contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function OverviewScheduledScans")
    expect(source).toContain("from \"react\"")
  })

  it("keeps loading on the resolved scheduled scan table geometry", () => {
    expect(source).toContain("function OverviewScheduledScansTableLoadingState")
    expect(source).toContain('data-slot="overview-scheduled-scans-table-loading-state"')
    expect(source).toContain('owner: "overview-scheduled-scans"')
    expect(source).toContain("data={[]}")
    expect(source).toContain("columns={columns}")
    expect(source).toContain("loading")
    expect(source).toContain("loadingRowCount={getDataTableSkeletonRowCount(pagination.pageSize)}")
    expect(source).toContain("showQuickFilters={false}")
    expect(source).not.toContain("<DataTableSkeleton")
    expect(source).not.toContain("toolbarButtonCount={2}")
    expect(source).not.toContain("columns={3}")
    expect(source).not.toContain("function OverviewScheduledScansTableSkeleton")
    expect(source).not.toContain('data-slot="overview-scheduled-scans-table-skeleton"')
  })
})
