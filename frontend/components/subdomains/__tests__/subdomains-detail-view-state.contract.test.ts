import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/subdomains/subdomains-detail-view-state.ts"), "utf8")

describe("subdomains-detail-view-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useSubdomainsDetailViewState")
    expect(source).toContain("from \"react\"")
  })

  it("uses BusinessListQuery for target subdomain search sorting and token pagination", () => {
    expect(source).toContain("createBusinessListQuery")
    expect(source).toContain("applyBusinessListControlChange")
    expect(source).toContain("compileBusinessListFilter")
    expect(source).toContain("compileBusinessListOrderBy")
    expect(source).toContain("setBusinessListPage")
    expect(source).toContain("toggleBusinessListSorting")
    expect(source).toContain('search: { field: "dnsName", operator: "=" }')
    expect(source).toContain('dnsName: { orderBy: "dnsName", firstDirection: "asc" }')
    expect(source).toContain('createdAt: { orderBy: "createdAt", firstDirection: "desc" }')
    expect(source).toContain("SUBDOMAIN_DEFAULT_SORTING")
    expect(source).toContain("pageTokens")
    expect(source).toContain("const targetPageToken = hasCursorScopeChanged ? undefined : query.pageToken")
    expect(source).toContain("pageToken: targetPageToken")
    expect(source).toContain("orderBy: compiledOrderBy")
  })

  it("uses BusinessListQuery for scan subdomain search sorting and token pagination", () => {
    expect(source).toContain("useScanSubdomains")
    expect(source).toContain("scanQuery")
    expect(source).toContain("scanPageTokens")
    expect(source).toContain("const scanPageToken = hasCursorScopeChanged ? undefined : scanQuery.pageToken")
    expect(source).toContain("pageToken: scanPageToken")
    expect(source).toContain("useCursorPaginationScopeChange")
    expect(source).toContain("orderBy: compiledScanOrderBy")
    expect(source).toContain('sortingMode: "server"')
    expect(source).not.toContain("page: scanPagination.pageIndex + 1")
    expect(source).not.toContain("scanFilterQuery")
  })
})
