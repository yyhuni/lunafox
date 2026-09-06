import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/targets/targets-detail-view-sections.tsx"), "utf8")

describe("targets-detail-view-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function TargetsDetailViewLoadingState")
    expect(source).toContain("className")
    expect(source).toContain("dynamic(")
    expect(source).toContain("useDeferredInteractionMount")
    expect(source).toContain("loading: () => null")
  })

  it("keeps loading on the resolved organization targets table geometry", () => {
    expect(source).toContain("state: TargetsDetailViewState")
    expect(source).toContain("data={[]}")
    expect(source).toContain("columns={state.targetColumns}")
    expect(source).toContain("onAddNew={state.handleAddTarget}")
    expect(source).toContain("onBulkDelete={state.handleBulkDelete}")
    expect(source).toContain("pagination={state.pagination}")
    expect(source).toContain("cursorPaginationSummary={state.cursorPaginationSummary}")
    expect(source).toContain("paginationNavigation={state.paginationNavigation}")
    expect(source).toContain("loading")
    expect(source).toContain("loadingRowCount={getDataTableSkeletonRowCount(state.pagination.pageSize)}")
    expect(source).not.toContain("<DataTableSkeleton")
    expect(source).not.toContain("toolbarButtonCount={2}")
    expect(source).not.toContain("columns={4}")
  })

  it("keeps bulk target unlink confirmation summary-only by default", () => {
    expect(source).toContain("bulkDeleteDialogOpen")
    expect(source).toContain("bulkUnlinkTargetMessage")
    expect(source).toContain("confirmUnlink")
    expect(source).not.toContain("state.selectedTargets.map")
  })
})
