import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ip-addresses/ip-addresses-data-table.tsx"), "utf8")

describe("ip-addresses-data-table contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function IPAddressesDataTable")
    expect(source).toContain("BusinessListDataTable")
  })

  it("uses the standard simple search toolbar instead of SmartFilter", () => {
    expect(source).toContain("SimpleSearchToolbar")
    expect(source).toContain("searchPlaceholder?: string")
    expect(source).toContain('placeholder={searchPlaceholder ?? tActions("searchIPOrHost")}')
    expect(source).not.toContain("SmartFilterBusinessListDataTable")
    expect(source).not.toContain("IP_ADDRESS_FILTER_EXAMPLES")
    expect(source).not.toContain("SmartFilterInput")
  })

  it("keeps multi-select port faceted filter wired beside default search", () => {
    expect(source).toContain("DataTableFacetedFilter")
    expect(source).toContain("DataTableFacetedFilterGroup")
    expect(source).toContain("portFilter?: string[]")
    expect(source).toContain("onPortFilterChange?: (values: string[]) => void")
    expect(source).toContain("title={translatedFields.port.label}")
    expect(source).toContain("values={portFilter}")
    expect(source).toContain("onValuesChange={onPortFilterChange}")
    expect(source).toContain("hasSelectedValues={portFilter.length > 0}")
    expect(source).toContain("onReset={() => onPortFilterChange?.([])}")
    expect(source).toContain("ui={{")
    expect(source).toContain("toolbarLeft,")
    expect(source).not.toContain("values.slice(-1)")
  })

  it("uses backend-backed port options instead of current-page row faceting", () => {
    expect(source).toContain("portOptions?: Array<DataTableFacetedFilterOption<string>>")
    expect(source).toContain("mergeSelectedFilterOptions")
    expect(source).toContain("const optionByValue = new Map(options.map((option) => [option.value, option]))")
    expect(source).toContain("for (const value of selected)")
    expect(source).toContain("optionByValue.set(trimmed, { value: trimmed, label: trimmed })")
    expect(source).toContain("options={mergedPortOptions}")
    expect(source).not.toContain("for (const item of data)")
    expect(source).not.toContain("for (const port of item.ports)")
    expect(source).not.toContain("data.flatMap")
    expect(source).not.toContain("getFacetedUniqueValues")
  })

  it("allows migrated IP address tables to run in server sorting mode", () => {
    expect(source).toContain("sortingMode?: DataTableSortingMode")
    expect(source).toContain('sortingMode = "none"')
    expect(source).toContain("sorting?: SortingState")
    expect(source).toContain("onSortingChange?: (sorting: SortingState) => void")
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
    expect(source).toContain("initialLoadingToolbarFilterCount: 1")
  })
})
