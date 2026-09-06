import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/vulnerabilities/vulnerabilities-vertical-view.tsx"), "utf8")

describe("vulnerabilities-vertical-view contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function VulnerabilitiesVerticalView")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("renders the shared vulnerabilities table and opens details in a scan-history-style side drawer", () => {
    expect(source).toContain("VulnerabilitiesDataTable")
    expect(source).toContain("VulnerabilityStatCards")
    expect(source).toContain("VulnerabilityStatCardsLoadingState")
    expect(source).toContain("VulnerabilityDetailDrawer")
    expect(source).toContain('from "./vulnerability-detail-drawer"')
    expect(source).toContain("counts={state.severityCounts}")
    expect(source).toContain("severityCounts={state.severityCounts}")
    expect(source).toContain("sourceFilter={state.sourceFilter}")
    expect(source).toContain("onSourceFilterChange={state.handleSourceFilterChange}")
    expect(source).toContain("sourceOptions={state.sourceOptions}")
    expect(source).toContain("vulnTypeFilter={state.vulnTypeFilter}")
    expect(source).toContain("onVulnTypeFilterChange={state.handleVulnTypeFilterChange}")
    expect(source).toContain("vulnTypeOptions={state.vulnTypeOptions}")
    expect(source).toContain('sortingMode="server"')
    expect(source).toContain("sorting={state.sorting}")
    expect(source).toContain("onSortingChange={state.handleSortingChange}")
    expect(source).toContain("totalCount={state.totalCount}")
    expect(source).toContain("onRowClick={handleSelectVulnerability}")
    expect(source).toContain("onBulkDelete={state.handleOpenBulkDeleteDialog}")
    expect(source).toContain("onOpenChange={handleDetailOpenChange}")
    expect(source).toContain("deferredInteractionUnmountDelayMs")
    expect(source).toContain("useDeferredInteractionMount(Boolean(activeVulnerability)")
    expect(source).toContain("shouldMountDetailDrawer ?")
    expect(source).toContain("useVulnerabilitiesDetailViewState({ scanId, targetId })")
    expect(source).not.toContain("VulnerabilitiesVerticalViewSkeleton")
    expect(source).not.toContain("VulnerabilitiesReviewTabsSkeleton")
    expect(source).not.toContain("VulnerabilitiesDataTableSkeleton")
    expect(source).not.toContain("SheetContent")
    expect(source).not.toContain("<DetailDrawer")
    expect(source).not.toContain("useVerticalResize")
    expect(source).not.toContain("VerticalResizeHandle")
    expect(source).not.toContain("VulnerabilityVerticalDetail")
    expect(source).not.toContain("<DataTableSkeleton")
  })

  it("keeps the loading skeleton geometry aligned with the resolved vulnerability list", () => {
    expect(source).toContain('className="flex min-h-0 flex-1 flex-col gap-4 md:gap-6"')
    expect(source).toContain("<VulnerabilityStatCardsLoadingState />")
    expect(source).not.toContain("VulnerabilityStatCardsSkeleton")
    expect(source).toContain("<VulnerabilitiesDataTable")
    expect(source).toContain("loading")
    expect(source).toContain("const loadingRowCount = getDataTableSkeletonRowCount(state.pagination.pageSize)")
    expect(source).toContain('getLoadingStructureSlotAttributes("vulnerabilities-severity-summary")')
    expect(source).toContain("loadingRowCount={loadingRowCount}")
    expect(source.match(/stableSurfaceRowCount={loadingRowCount}/g)).toHaveLength(2)
    expect(source).not.toContain("fillAvailableHeight")
    expect(source).not.toContain('owner="vulnerabilities-table-view"')
    expect(source).not.toContain('toolbarHeight="tall"')
  })

  it("keeps the same named shared-table regions for loading and resolved vulnerability states", () => {
    expect(source).toContain("const VULNERABILITIES_LOADING_SLOTS")
    expect(source).toContain('toolbar: "vulnerabilities-table-toolbar"')
    expect(source).toContain('body: "vulnerabilities-rows"')
    expect(source).toContain('pagination: "vulnerabilities-pagination"')
    expect(source.match(/loadingSlots={VULNERABILITIES_LOADING_SLOTS}/g)).toHaveLength(2)
  })

  it("keeps the workspace height chain during loading and resolved states", () => {
    expect(source).toContain('className="flex min-h-0 flex-1 flex-col"')
    expect(source).toContain('skeletonClassName="flex min-h-0 flex-1 flex-col"')
    expect(source).toContain('contentClassName="flex min-h-0 flex-1 flex-col"')
  })
})
