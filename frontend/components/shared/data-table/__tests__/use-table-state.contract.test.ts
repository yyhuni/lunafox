import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/data-table/use-table-state.ts"), "utf8")

describe("use-table-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useTableState")
    expect(source).toContain("from \"react\"")
  })

  it("normalizes column width floors so sortable header titles can raise narrow min sizes", () => {
    expect(source).toContain("getColumnHeaderMinWidthPx")
    expect(source).toContain("getBadgeMinWidthPx")
    expect(source).toContain("singleBadge")
    expect(source).toContain("singleBadgeValues")
    expect(source).toContain('const resolvedSortingMode = sortingMode ?? (paginationInfo ? "none" : "client")')
    expect(source).toContain("enableSorting !== false")
    expect(source).toContain("resolvedSortingMode !== \"server\" || Boolean(columnDef.meta?.orderBy && columnDef.meta?.serverSortPerformance)")
    expect(source).toContain("manualSorting: resolvedSortingMode === \"server\"")
    expect(source).toContain("getSortedRowModel: resolvedSortingMode === \"client\" ? getSortedRowModel() : undefined")
    expect(source).toContain("Math.max(declaredSize, normalizedMinSize)")
    expect(source).toContain("Math.max(declaredMinSize, headerMinWidthPx, singleBadgeFloorPx)")
  })

  it("keeps computed column sizing but does not enable user column resizing", () => {
    expect(source).toContain("const [columnSizing, setColumnSizing]")
    expect(source).not.toContain("enableColumnResizing")
    expect(source).not.toContain("columnResizeMode")
    expect(source).not.toContain("onColumnSizingChange")
  })

  it("keeps TanStack faceting available only as table state plumbing, not backend-paginated option ownership", () => {
    expect(source).toContain("getFacetedRowModel: getFacetedRowModel()")
    expect(source).toContain("getFacetedUniqueValues: getFacetedUniqueValues()")
    expect(source).toContain('const resolvedSortingMode = sortingMode ?? (paginationInfo ? "none" : "client")')
    expect(source).toContain("manualPagination: !!paginationInfo")
    expect(source).not.toContain("getFacetedUniqueValues().entries")
    expect(source).not.toContain("getFacetedUniqueValues().keys")
    expect(source).not.toContain("facetedFilterOptions")
    expect(source).not.toContain("backendFilterOptions")
  })
})
