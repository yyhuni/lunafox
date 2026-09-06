import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/vulnerabilities/vulnerabilities-data-table.tsx"), "utf8")

describe("vulnerabilities-data-table contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function VulnerabilitiesDataTable")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses shared tab count badges for review filters", () => {
    expect(source).toContain("TabsCountBadge")
    expect(source).not.toContain("rounded-full text-xs")
  })

  it("places review filters above the business-list table instead of in the right toolbar", () => {
    expect(source).toContain("function VulnerabilityReviewTabs")
    expect(source).toContain('variant="content"')
    expect(source).toContain('"space-y-4"')
    expect(source).not.toContain('className="mb-0 gap-0"')
    expect(source).toContain("totalCount?: number")
    expect(source).not.toContain("rightToolbarContent")
    expect(source).not.toContain("toolbarRight:")
  })

  it("keeps the controlled variant on fixed semantic widths with paired primary expand columns", () => {
    expect(source).toContain("BusinessListDataTable")
    expect(source).toContain('expandColumnIds: ["vulnType", "url"]')
    expect(source).not.toContain("<UnifiedDataTable")
  })

  it("allows route owners to open row details without rebuilding the table shell", () => {
    expect(source).toContain("onRowClick?: (vulnerability: Vulnerability) => void")
    expect(source).toContain("onRowClick(row as Vulnerability)")
    expect(source).toContain("getRowActionLabel")
    expect(source).toContain("tVuln(\"openDetailFor\"")
    expect(source).toContain("onRowClick,")
  })

  it("renders selected-row actions through the shared selected action bar", () => {
    expect(source).toContain("SelectedRowActionBar")
    expect(source).toContain("const DeleteIcon = semanticIcons.action.delete")
    expect(source).toContain("onBulkMarkAsReviewed || onBulkMarkAsPending || onBulkDelete")
    expect(source).toContain('group: "status"')
    expect(source).toContain('group: "danger"')
    expect(source).toContain("onClick: onBulkDelete")
    expect(source).toContain("tActions(\"delete\")")
    expect(source).toContain("onClearSelection")
    expect(source).not.toContain('role="toolbar"')
    expect(source).not.toContain("border-destructive/45")
    expect(source).not.toContain("bulkDeleteLabel")
  })

  it("groups severity, source, and vulnerability type in the shared filter panel", () => {
    expect(source).toContain('from "@/components/shared/data-table/faceted-filter"')
    expect(source).toContain("DataTableFacetPanel,")
    expect(source).toContain("type DataTableFacetPanelFacet,")
    expect(source).toContain("export type SeverityFilter = VulnerabilitySeverity[]")
    expect(source).toContain("export type VulnerabilityTextFilter = string[]")
    expect(source).toContain("severityFilter = []")
    expect(source).toContain("sourceFilter = []")
    expect(source).toContain("vulnTypeFilter = []")
    expect(source).toContain("severityCounts?: VulnerabilitySeverityCounts")
    expect(source).toContain("sourceOptions?: Array<DataTableFacetedFilterOption<string>>")
    expect(source).toContain("vulnTypeOptions?: Array<DataTableFacetedFilterOption<string>>")
    expect(source).toContain("const vulnerabilityFacetPanelItems: DataTableFacetPanelFacet[] = []")
    expect(source).toContain('id: "severity"')
    expect(source).toContain('label: tVuln("severityLabel")')
    expect(source).toContain('id: "source"')
    expect(source).toContain('label: tColumns("vulnerability.source")')
    expect(source).toContain("options: mergedSourceOptions")
    expect(source).toContain('id: "vulnType"')
    expect(source).toContain('label: tColumns("vulnerability.vulnType")')
    expect(source).toContain("options: mergedVulnTypeOptions")
    expect(source).toContain("mergeSelectedFilterOptions")
    expect(source).toContain("const activeFacetFilterCount = severityFilter.length + sourceFilter.length + vulnTypeFilter.length")
    expect(source).toContain("<DataTableFacetPanel")
    expect(source).toContain('title={tDataTable("filter")}')
    expect(source).toContain("facets={vulnerabilityFacetPanelItems}")
    expect(source).toContain("activeCount={activeFacetFilterCount}")
    expect(source).not.toContain("<DataTableFacetedFilter\n")
    expect(source).not.toContain("<DataTableFacetedFilterGroup")
    expect(source).toContain('emptyLabel: tVuln("reviewStatus.all")')
    expect(source).toContain('const tDataTable = useTranslations("dataTable")')
    expect(source).toContain('clearLabel: tDataTable("clearFilter")')
    expect(source).not.toContain("<SelectTrigger size=\"sm\">")
    expect(source).not.toContain("<SelectContent width=\"content-fit\">")
  })

  it("uses the scan-history inline search toolbar pattern before faceted filters", () => {
    expect(source).toContain('from "@/components/shared/data-table/simple-search-toolbar"')
    expect(source).toContain('from "@/components/shared/data-table/use-simple-search"')
    expect(source).toContain("<SimpleSearchToolbar")
    expect(source).toContain('placeholder={tActions("searchURL")}')
    expect(source).toContain("handleSearchInputChange")
    expect(source).toContain("commitNow: commitSearch")
    expect(source).toContain("onChange={handleSearchInputChange}")
    expect(source).toContain("onSubmit={commitSearch}")
    expect(source).not.toContain("handleSearchSubmit")
    expect(source).not.toContain("setLocalSearchValue")
    expect(source).not.toContain('inputClassName="w-full sm:w-72 lg:w-80"')
    expect(source).not.toContain("SmartFilterInput")
    expect(source).not.toContain("VULNERABILITY_FILTER_EXAMPLES")
  })

  it("attaches severity summary counts to faceted severity options", () => {
    expect(source).toContain("type DataTableFacetedFilterOption")
    expect(source).toContain("const severityOptions: Array<DataTableFacetedFilterOption<VulnerabilitySeverity>>")
    expect(source).toContain("count: severityCounts?.critical ?? 0")
    expect(source).toContain("count: severityCounts?.high ?? 0")
    expect(source).toContain("count: severityCounts?.medium ?? 0")
    expect(source).toContain("count: severityCounts?.low ?? 0")
    expect(source).toContain("count: severityCounts?.info ?? 0")
  })

  it("keeps the search width stable when faceted severity badges appear", () => {
    expect(source).toContain('className="flex w-full flex-wrap items-center gap-2 sm:flex-1"')
    expect(source).not.toContain('inputClassName="w-full sm:w-72 lg:w-80"')
    expect(source).not.toContain('className="flex-1 max-w-md"')
  })

  it("uses the shared initial-loading presentation for cold list skeletons", () => {
    expect(source).toContain('loadingPresentation: loading ? "initial" : undefined')
    expect(source).toContain('from "@/components/ui/table"')
    expect(source).toContain("TABLE_DENSE_ROW_ESTIMATED_HEIGHT_PX")
    expect(source).toContain("loadingRowHeightEstimate: TABLE_DENSE_ROW_ESTIMATED_HEIGHT_PX")
  })

  it("forwards the parent-owned geometry slots to the shared table shell", () => {
    expect(source).toContain("loadingSlots?: DataTableLoadingSlots")
    expect(source).toContain("stableSurfaceRowCount?: number")
    expect(source).toContain("stableSurfaceRowCount,")
    expect(source).toContain("loadingSlots,")
    expect(source).toContain("loadingSlots,")
  })

  it("keeps review tabs and the shared table in natural flow", () => {
    expect(source).toContain('className="space-y-4"')
    expect(source).not.toContain("fillAvailableHeight")
  })
})
