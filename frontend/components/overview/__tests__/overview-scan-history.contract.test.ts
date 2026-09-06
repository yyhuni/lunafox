import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/overview/overview-scan-history.tsx"), "utf8")

describe("overview-scan-history contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function OverviewScanHistory")
    expect(source).toContain("from \"react\"")
  })

  it("matches the hidden-toolbar preview table loading chrome", () => {
    expect(source).toContain("function OverviewScanHistoryTableLoadingState")
    expect(source).toContain('data-slot="overview-scan-history-table-loading-state"')
    expect(source).toContain('owner: "overview-scan-history"')
    expect(source).toContain("data={[]}")
    expect(source).toContain("columns={columns}")
    expect(source).toContain("hideToolbar")
    expect(source).toContain("hidePagination")
    expect(source).toContain("loading")
    expect(source).toContain("loadingRowCount={getDataTableSkeletonRowCount(pagination.pageSize)}")
    expect(source).not.toContain("<DataTableSkeleton")
    expect(source).not.toContain("withSearch={false}")
    expect(source).not.toContain("toolbarButtonCount={0}")
    expect(source).not.toContain("withPagination={false}")
    expect(source).not.toContain("function OverviewScanHistoryTableSkeleton")
    expect(source).not.toContain('data-slot="overview-scan-history-table-skeleton"')
  })
})
