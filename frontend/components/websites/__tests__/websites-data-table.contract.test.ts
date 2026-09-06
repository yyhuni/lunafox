import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/websites/websites-data-table.tsx"), "utf8")

describe("websites-data-table contract", () => {
  it("keeps loading rows on the shared dense table height estimate", () => {
    expect(source).toContain("TABLE_DENSE_ROW_ESTIMATED_HEIGHT_PX")
    expect(source).toContain("loadingRowHeightEstimate: TABLE_DENSE_ROW_ESTIMATED_HEIGHT_PX")
  })

  it("preserves current source markers", () => {
    expect(source).toContain("export function WebSitesDataTable")
    expect(source).toContain("BusinessListDataTable")
  })

  it("maps cold route loading to the shared initial table presentation", () => {
    expect(source).toContain("initialLoading?: boolean")
    expect(source).toContain('loadingPresentation: initialLoading ? "initial" : "rows"')
    expect(source).toContain("initialLoadingToolbarFilterCount: 1")
  })

  it("hard cuts SmartFilter syntax UI from the website table", () => {
    expect(source).not.toContain("SmartFilterBusinessListDataTable")
    expect(source).not.toContain("FilterField")
    expect(source).not.toContain("WEBSITE_FILTER_EXAMPLES")
    expect(source).not.toContain("filterExamples")
    expect(source).not.toContain("filterValue")
    expect(source).not.toContain("onFilterChange")
    expect(source).toContain("SimpleSearchToolbar")
    expect(source).toContain("searchValue?: string")
    expect(source).toContain("onSearchChange?: (value: string) => void")
    expect(source).toContain('placeholder={tActions("searchURL")}')
  })

  it("renders structured website facets in the aggregate filter panel and keeps server-side sorting", () => {
    expect(source).toContain("DataTableFacetPanel")
    expect(source).toContain("DataTableFacetPanelFacet")
    expect(source).toContain("websiteFacetPanelItems")
    expect(source).toContain("facets={websiteFacetPanelItems}")
    expect(source).toContain("activeCount={activeFilterCount}")
    expect(source).toContain("const activeFilterCount = statusCodeFilter.length")
    expect(source).not.toContain("DataTableActiveFilterList")
    expect(source).not.toContain("createActiveFilterTokens")
    expect(source).not.toContain("resetFacetFilters")
    expect(source).not.toContain('useTranslations("filter")')
    expect(source).not.toContain('label={tFilter("groups.activeFilters")}')
    expect(source).not.toContain('onClear={resetFacetFilters}\n          clearLabel={tDataTable("clearFilter")}\n          facets={websiteFacetPanelItems}')
    expect(source).not.toContain("DataTableFacetedFilterGroup")
    expect(source).toContain("statusCodeFilter")
    expect(source).toContain("techFilter")
    expect(source).toContain("webserverFilter")
    expect(source).toContain("contentTypeFilter")
    expect(source).toContain("vhostFilter")
    expect(source).toContain("sortingMode")
    expect(source).toContain("onSortingChange")
  })

  it("uses backend-backed filter options instead of current-page row faceting", () => {
    expect(source).toContain("statusCodeOptions?: Array<DataTableFacetedFilterOption<string>>")
    expect(source).toContain("techOptions?: Array<DataTableFacetedFilterOption<string>>")
    expect(source).toContain("webserverOptions?: Array<DataTableFacetedFilterOption<string>>")
    expect(source).toContain("contentTypeOptions?: Array<DataTableFacetedFilterOption<string>>")
    expect(source).toContain("vhostOptions?: Array<DataTableFacetedFilterOption<string>>")
    expect(source).toContain("mergeSelectedFilterOptions")
    expect(source).toContain("const optionByValue = new Map(options.map((option) => [option.value, option]))")
    expect(source).toContain("for (const value of selected)")
    expect(source).toContain("optionByValue.set(trimmed, { value: trimmed, label: trimmed })")
    expect(source).not.toContain("data.flatMap((item) => item.tech")
    expect(source).not.toContain("data.map((item) => item.webserver")
    expect(source).not.toContain("data.map((item) => item.contentType")
    expect(source).not.toContain("data.map((item) => item.statusCode")
    expect(source).not.toContain("data.map((item) => item.vhost")
    expect(source).not.toContain("getFacetedUniqueValues")
  })

  it("keeps the website default table on semantic-width business-list behavior with shared column controls", () => {
    expect(source).toContain("VisibilityState")
    expect(source).toContain("columnVisibility?: VisibilityState")
    expect(source).toContain("onColumnVisibilityChange?: (visibility: VisibilityState) => void")
    expect(source).toContain("columnVisibility,")
    expect(source).toContain("onColumnVisibilityChange,")
    expect(source).toContain("showColumnVisibility: true")
    expect(source).not.toContain("expandColumnIds")
  })

  it("delegates website detail activation to shared row interaction semantics", () => {
    expect(source).toContain("onRowClick?: (website: WebSite) => void")
    expect(source).toContain("onRowClick,")
    expect(source).toContain("behavior={{\n        onRowClick: onRowClick")
    expect(source).toContain("getRowActionLabel")
  })

  it("exposes shared table loading controls for stateful page loading", () => {
    expect(source).toContain("loading?: boolean")
    expect(source).toContain("loadingRowCount?: number")
    expect(source).toContain("loading = false")
    expect(source).toContain("loadingRowCount,")
    expect(source).toContain("loading,")
    expect(source).toContain("loadingRowCount,")
  })
})
