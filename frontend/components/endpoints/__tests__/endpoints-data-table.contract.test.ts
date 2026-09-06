import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/endpoints/endpoints-data-table.tsx"), "utf8")

describe("endpoints-data-table contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function EndpointsDataTable")
    expect(source).toContain("BusinessListDataTable")
  })

  it("hard cuts SmartFilter syntax UI from the endpoint table", () => {
    expect(source).not.toContain("SmartFilterBusinessListDataTable")
    expect(source).not.toContain("FilterField")
    expect(source).not.toContain("ENDPOINT_FILTER_EXAMPLES")
    expect(source).not.toContain("filterExamples")
    expect(source).not.toContain("filterValue")
    expect(source).not.toContain("onFilterChange")
    expect(source).toContain("SimpleSearchToolbar")
    expect(source).toContain("searchValue?: string")
    expect(source).toContain("onSearchChange?: (value: string) => void")
    expect(source).toContain('placeholder={tActions("searchURL")}')
  })

  it("renders structured endpoint facets in the shared aggregate panel and keeps server-side sorting", () => {
    expect(source).toContain("DataTableFacetPanel")
    expect(source).toContain("DataTableFacetPanelFacet")
    expect(source).toContain("endpointFacetPanelItems")
    expect(source).toContain("facets={endpointFacetPanelItems}")
    expect(source).toContain("activeCount={activeFilterCount}")
    expect(source).toContain("const activeFilterCount = statusCodeFilter.length")
    expect(source).not.toContain("DataTableActiveFilterList")
    expect(source).not.toContain("createActiveFilterTokens")
    expect(source).not.toContain("resetFacetFilters")
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
    expect(source).not.toContain("data.flatMap((item) => item.tech")
    expect(source).not.toContain("data.map((item) => item.webserver")
    expect(source).not.toContain("data.map((item) => item.contentType")
    expect(source).not.toContain("getFacetedUniqueValues")
  })

  it("keeps the endpoint default table on semantic-width business-list behavior with shared column controls", () => {
    expect(source).toContain("VisibilityState")
    expect(source).toContain("columnVisibility?: VisibilityState")
    expect(source).toContain("onColumnVisibilityChange?: (visibility: VisibilityState) => void")
    expect(source).toContain("columnVisibility,")
    expect(source).toContain("onColumnVisibilityChange,")
    expect(source).toContain("showColumnVisibility: true")
    expect(source).not.toContain("expandColumnIds")
  })

  it("keeps sparse pages compact instead of padding empty row slots", () => {
    expect(source).not.toContain("preserveEmptyStatePageSlots")
  })

  it("exposes shared table loading controls for stateful page loading", () => {
    expect(source).toContain("loading?: boolean")
    expect(source).toContain("initialLoading?: boolean")
    expect(source).toContain("loadingRowCount?: number")
    expect(source).toContain("loading = false")
    expect(source).toContain("initialLoading = false")
    expect(source).toContain("loadingRowCount,")
    expect(source).toContain("loading,")
    expect(source).toContain('loadingPresentation: initialLoading ? "initial" : "rows"')
    expect(source).toContain("loadingRowCount,")
  })
})
