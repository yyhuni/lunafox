import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/vulnerabilities/vulnerabilities-detail-view-sections.tsx"), "utf8")
const dataTableSource = readFileSync(path.resolve(process.cwd(), "components/vulnerabilities/vulnerabilities-data-table.tsx"), "utf8")

describe("vulnerabilities-detail-view-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function VulnerabilitiesDetailViewLoadingState")
    expect(source).toContain("export function VulnerabilitiesDetailViewRouteFallback")
    expect(source).toContain("className")
  })

  it("keeps row detail ownership outside the table content", () => {
    expect(source).not.toContain("InteractionLoadingDialog")
    expect(source).not.toContain("VulnerabilityDetailDialog")
    expect(source).not.toContain("detail-dialog")
    expect(source).not.toContain("fillAvailableHeight")
  })

  it("uses the vulnerabilities-specific loading structure for review tabs and table geometry", () => {
    expect(source).toContain("state: VulnerabilitiesDetailViewState")
    expect(source).toContain("<VulnerabilitiesDataTable")
    expect(source).toContain("loading")
    expect(source).toContain("loadingRowCount={rowCount}")
    expect(dataTableSource).toContain("initialLoadingToolbarFilterCount: vulnerabilityFacetPanelItems.length > 0 ? 1 : 0")
    expect(source).toContain("function VulnerabilitiesRouteFallbackReviewTabs")
    expect(source).toContain("function VulnerabilitiesRouteFallbackTable")
    expect(source).toContain("<VulnerabilitiesRouteFallbackReviewTabs />")
    expect(source).toContain("<VulnerabilitiesRouteFallbackTable rows={rowCount} />")
    expect(source).not.toContain('from "./vulnerabilities-detail-view-skeleton"')
    expect(source).not.toContain('from "./vulnerabilities-data-table-skeleton"')
    expect(source).not.toContain("return <VulnerabilitiesDetailViewSkeleton rowCount={rowCount} />")
    expect(source).not.toContain('owner="vulnerabilities-detail-view"')
    expect(source).not.toContain("<DataTableSkeleton")
    expect(source).not.toContain('from "@/components/shared/loading/data-table-skeleton"')
  })

  it("keeps the route fallback inside the real sections owner", () => {
    expect(source).toContain("const vulnerabilityFallbackTranslations")
    expect(source).toContain("createVulnerabilityColumns")
    expect(source).toContain("VULNERABILITIES_ROUTE_FALLBACK_PAGE_SIZE")
    expect(source).toContain("pageSize: VULNERABILITIES_ROUTE_FALLBACK_PAGE_SIZE")
    expect(source).not.toContain("pageSize: rows")
    expect(source).toContain('<TabsList variant="content" size="sm"')
    expect(source).toContain("<TabsCountBadge>")
    expect(source).toContain('tVulnerabilities("reviewStatus.allVulnerabilities")')
    expect(source).toContain('tVulnerabilities("reviewStatus.pending")')
    expect(source).toContain('tVulnerabilities("reviewStatus.reviewed")')
    expect(source).not.toContain("labelWidth")
    expect(source).not.toContain('from "@/components/shared/loading/action-skeleton"')
    expect(source).not.toContain('from "@/components/shared/loading/search-toolbar-skeleton"')
    expect(source).not.toContain('from "@/components/shared/loading/select-shell-skeleton"')
    expect(source).not.toContain('from "@/components/shared/loading/compact-pagination-skeleton"')
  })

  it("passes the reviewed bulk delete confirmation entrypoint to the selected-row toolbar", () => {
    expect(source).toContain("onBulkDelete={state.isReadOnly ? undefined : state.handleOpenBulkDeleteDialog}")
    expect(source).toContain("bulkDeleteDialogOpen")
    expect(source).toContain("confirmBulkDelete")
  })

  it("passes row detail activation through to the shared vulnerabilities table", () => {
    expect(source).toContain("onRowClick?: (vulnerability: Vulnerability) => void")
    expect(source).toContain("onRowClick={onRowClick}")
  })

  it("passes severity summary counts into the shared faceted severity filter", () => {
    expect(source).toContain("severityCounts={state.severityCounts}")
  })

  it("passes source and vulnerability type facets after backend options are available", () => {
    expect(source).toContain("sourceFilter={state.sourceFilter}")
    expect(source).toContain("onSourceFilterChange={state.handleSourceFilterChange}")
    expect(source).toContain("sourceOptions={state.sourceOptions}")
    expect(source).toContain("vulnTypeFilter={state.vulnTypeFilter}")
    expect(source).toContain("onVulnTypeFilterChange={state.handleVulnTypeFilterChange}")
    expect(source).toContain("vulnTypeOptions={state.vulnTypeOptions}")
    expect(source).toContain('sortingMode="server"')
    expect(source).toContain("sorting={state.sorting}")
    expect(source).toContain("onSortingChange={state.handleSortingChange}")
  })

  it("keeps bulk vulnerability delete confirmation summary-only by default", () => {
    expect(source).toContain("bulkDeleteVulnMessage")
    expect(source).toContain("confirmDelete")
    expect(source).not.toContain("state.selectedVulnerabilities.map")
  })
})
