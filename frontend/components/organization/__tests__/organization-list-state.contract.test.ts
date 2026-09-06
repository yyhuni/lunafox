import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/organization-list-state.ts"), "utf8")

describe("organization-list-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useOrganizationListState")
    expect(source).toContain("from \"react\"")
  })

  it("tracks the selected organization for the detail drawer", () => {
    expect(source).toContain("organizationToView")
    expect(source).toContain("handleViewDetail")
    expect(source).toContain("handleDetailOpenChange")
    expect(source).toContain("setOrganizationToView(null)")
  })

  it("uses the shared business-list query controls for backend search sorting and pagination", () => {
    expect(source).toContain("createBusinessListQuery")
    expect(source).toContain("applyBusinessListControlChange")
    expect(source).toContain("compileBusinessListFilter")
    expect(source).toContain("compileBusinessListOrderBy")
    expect(source).toContain("setBusinessListPage")
    expect(source).toContain("toggleBusinessListSorting")
    expect(source).toContain("ORGANIZATION_SORTABLE_FIELDS")
    expect(source).toContain('displayName: { orderBy: "displayName", firstDirection: "asc" }')
    expect(source).toContain('createdAt: { orderBy: "createdAt", firstDirection: "desc" }')
    expect(source).toContain('search: { field: "displayName", operator: "=" }')
    expect(source).not.toContain("useSearchState")
    expect(source).not.toContain("page: pagination.pageIndex + 1")
  })

  it("resets token pagination when search sorting or page size changes", () => {
    expect(source).toContain("pageTokens")
    expect(source).toContain("resetPaging")
    expect(source).toContain("data?.nextPageToken")
    expect(source).toContain("handleSortingChange")
    expect(source).toContain("SortingState")
    expect(source).toContain("pageToken: query.pageToken")
    expect(source).toContain("orderBy: compiledOrderBy")
  })
})
