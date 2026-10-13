import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/targets/targets-data-table.tsx"), "utf8")

describe("targets-data-table contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function TargetsDataTable")
    expect(source).toContain("from \"@/components/shared/data-table/faceted-filter\"")
    expect(source).toContain("BusinessListDataTable")
  })

  it("reuses one shared filter panel for type and scanned status", () => {
    expect(source).toContain("const hasQueryControls = Boolean(onSearch || onTypeFilterChange || onScannedFilterChange)")
    expect(source).toContain("toolbarLeft: hasQueryControls ?")
    expect(source).toContain("<DataTableFacetPanel")
    expect(source).toContain("typeFilter?: TargetType[]")
    expect(source).toContain("onTypeFilterChange?: (value: TargetType[]) => void")
    expect(source).toContain("onSearch?: (value: string) => void")
    expect(source).toContain("values: typeFilter ?? []")
    expect(source).toContain("scannedFilter?: OrganizationTargetScannedFilter")
    expect(source).toContain('id: "type"')
    expect(source).toContain('id: "scanned"')
    expect(source).toContain('label: state.tTarget("scannedFilter.title")')
    expect(source).toContain('value: "true"')
    expect(source).toContain('value: "false"')
    expect(source).toContain('label: state.tColumns("common.type")')
    expect(source).toContain('emptyLabel: state.tCommon("actions.all")')
    expect(source).toContain('label: state.tTarget("types.domain")')
    expect(source).toContain('label: state.tTarget("types.ip")')
    expect(source).toContain('label: state.tTarget("types.cidr")')
    expect(source).toContain('clearLabel: state.tDataTable("clearFilter")')
    expect(source).not.toContain("TargetTypeFacetedFilter")
    expect(source).not.toContain("DataTableFacetedFilterGroup")
    expect(source).not.toContain("TargetTypeFilterSelect")
  })

  it("pins only the real primary target-name column as the expand owner for stable width allocation", () => {
    expect(source).toContain('expandColumnIds: ["name"]')
    expect(source).not.toContain('expandColumnIds: ["name", "organizations"]')
    expect(source).not.toContain('from "@/components/shared/data-table/unified-data-table"')
    expect(source).not.toContain('enableAutoColumnSizing: true')
  })

  it("hides the column visibility control for the organization target table", () => {
    expect(source).toContain("showColumnVisibility: false")
  })

  it("forwards the shared stable first-frame row reservation", () => {
    expect(source).toContain("stableSurfaceRowCount?: number")
    expect(source).toContain("stableSurfaceRowCount,")
  })

  it("uses the shared cursor pagination surface for backend target collections", () => {
    expect(source).toContain("CursorPaginationNavigation")
    expect(source).toContain("CursorPaginationSummary")
    expect(source).toContain("cursorPaginationSummary?: CursorPaginationSummary")
    expect(source).toContain("paginationNavigation?: CursorPaginationNavigation")
    expect(source).toContain("cursorPaginationSummary,")
    expect(source).toContain("paginationNavigation,")
    expect(source).not.toContain("paginationInfo?: PaginationInfo")
    expect(source).not.toContain("setPagination?: Dispatch")
  })

  it("keeps embedded organization target tables in natural-height mode", () => {
    expect(source).not.toContain("fillAvailableHeight")
  })
})
