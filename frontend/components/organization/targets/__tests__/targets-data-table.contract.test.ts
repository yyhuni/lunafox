import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/targets/targets-data-table.tsx"), "utf8")

describe("targets-data-table contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function TargetsDataTable")
    expect(source).toContain("from \"@/components/shared/data-table/target-type-filter-select\"")
    expect(source).toContain("BusinessListDataTable")
  })

  it("reuses the shared target-type faceted multi-select filter", () => {
    expect(source).toContain("const hasQueryControls = Boolean(onSearch || onTypeFilterChange)")
    expect(source).toContain("toolbarLeft: hasQueryControls ?")
    expect(source).toContain("TargetTypeFacetedFilter")
    expect(source).toContain("typeFilter?: TargetType[]")
    expect(source).toContain("onTypeFilterChange?: (value: TargetType[]) => void")
    expect(source).toContain("onSearch?: (value: string) => void")
    expect(source).toContain("values={typeFilter ?? []}")
    expect(source).toContain("onValuesChange={onTypeFilterChange}")
    expect(source).toContain('title={state.tColumns("common.type")}')
    expect(source).toContain("labels={{")
    expect(source).toContain('all: state.tCommon("actions.all")')
    expect(source).toContain('domain: state.tTarget("types.domain")')
    expect(source).toContain('ip: state.tTarget("types.ip")')
    expect(source).toContain('cidr: state.tTarget("types.cidr")')
    expect(source).toContain('clearLabel={state.tDataTable("clearFilter")}')
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
