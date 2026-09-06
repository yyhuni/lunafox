import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/directories/directories-data-table.tsx"), "utf8")

describe("directories-data-table contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function DirectoriesDataTable")
    expect(source).toContain("BusinessListDataTable")
  })

  it("hard cuts SmartFilter syntax UI from the directory table", () => {
    expect(source).not.toContain("SmartFilterBusinessListDataTable")
    expect(source).not.toContain("FilterField")
    expect(source).not.toContain("DIRECTORY_FILTER_EXAMPLES")
    expect(source).not.toContain("filterExamples")
    expect(source).not.toContain("filterValue")
    expect(source).not.toContain("onFilterChange")
    expect(source).toContain("SimpleSearchToolbar")
    expect(source).toContain("searchValue?: string")
    expect(source).toContain("onSearchChange?: (value: string) => void")
    expect(source).toContain('placeholder={tActions("searchURL")}')
  })

  it("renders structured directory facets and server-side sorting state", () => {
    expect(source).toContain("DataTableFacetPanel")
    expect(source).toContain("DataTableFacetPanelFacet")
    expect(source).toContain("const directoryFacetPanelItems: DataTableFacetPanelFacet[]")
    expect(source).toContain('title={tDataTable("filter")}')
    expect(source).toContain("activeCount={activeFilterCount}")
    expect(source).toContain("statusFilter")
    expect(source).toContain("contentTypeFilter")
    expect(source).toContain("sortingMode")
    expect(source).toContain("onSortingChange")
  })

  it("preserves selected filter values without deriving full options from current page rows", () => {
    expect(source).toContain("statusOptions?: Array<DataTableFacetedFilterOption<string>>")
    expect(source).toContain("contentTypeOptions?: Array<DataTableFacetedFilterOption<string>>")
    expect(source).toContain("mergeSelectedFilterOptions")
    expect(source).toContain("const optionByValue = new Map(options.map((option) => [option.value, option]))")
    expect(source).toContain("for (const value of selected)")
    expect(source).toContain("optionByValue.set(trimmed, { value: trimmed, label: trimmed })")
    expect(source).not.toContain("data.map((item) => item.status")
    expect(source).not.toContain("data.map((item) => item.contentType")
    expect(source).not.toContain("getFacetedUniqueValues")
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
    expect(source).not.toContain("loadingRowHeightEstimate")
    expect(source).not.toContain("TABLE_DENSE_ROW_ESTIMATED_HEIGHT_PX")
  })
})
