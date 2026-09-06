import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/target/all-targets-detail-view-state.ts"), "utf8")

describe("all-targets-detail-view-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useAllTargetsDetailViewState")
    expect(source).toContain("from \"react\"")
  })

  it("compiles search, type facets, sorting, and page tokens through BusinessListQuery", () => {
    expect(source).toContain("createBusinessListQuery")
    expect(source).toContain("applyBusinessListControlChange")
    expect(source).toContain("setBusinessListPage")
    expect(source).toContain("compileBusinessListFilter")
    expect(source).toContain("compileBusinessListOrderBy")
    expect(source).toContain("toggleBusinessListSorting")
    expect(source).toContain('search: { field: "displayName", operator: "=" }')
    expect(source).toContain('type: { field: "type", operator: "==" }')
    expect(source).toContain('displayName: { orderBy: "displayName", firstDirection: "asc" }')
    expect(source).toContain('createdAt: { orderBy: "createdAt", firstDirection: "desc" }')
    expect(source).toContain('lastScannedAt: { orderBy: "lastScannedAt", firstDirection: "desc" }')
    expect(source).toContain('TARGET_DEFAULT_SORTING')
    expect(source).toContain('pageTokens')
    expect(source).toContain('pageToken: query.pageToken')
    expect(source).toContain('orderBy: compiledOrderBy')
  })

  it("hard-cuts current-page target type filtering from production state", () => {
    expect(source).toContain("typeFilter")
    expect(source).toContain("handleTypeFilterChange")
    expect(source).not.toContain("const filteredTargets = React.useMemo")
    expect(source).not.toContain("typeFilter.includes(target.type)")
    expect(source).not.toContain("filteredTotalCount")
    expect(source).not.toContain("targets: filteredTargets")
    expect(source).not.toContain("totalCount: filteredTotalCount")
    expect(source).not.toContain("undefined,\n    searchQuery || undefined")
  })
})
