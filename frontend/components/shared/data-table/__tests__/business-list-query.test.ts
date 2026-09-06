import { describe, expect, it } from "vitest"

import {
  applyBusinessListControlChange,
  compileBusinessListFilter,
  compileBusinessListOrderBy,
  createBusinessListQuery,
  getCurrentCursorNextPageToken,
  getCursorPageTransition,
  getCursorPaginationNavigation,
  preserveRawURLSearchInput,
  setBusinessListPage,
  toggleBusinessListSorting,
  type BusinessListQuery,
} from "@/components/shared/data-table/business-list-query"

describe("business-list-query", () => {
  it("does not treat React Query placeholder data as the active cursor response", () => {
    expect(getCurrentCursorNextPageToken("page-two", true)).toBeUndefined()
    expect(getCurrentCursorNextPageToken("page-two", false)).toBe("page-two")
    expect(getCurrentCursorNextPageToken(undefined, false)).toBeUndefined()
  })

  it("derives cursor availability from cache keys and the active response token", () => {
    expect(getCursorPaginationNavigation({
      currentPage: 1,
      pageTokens: { 1: undefined },
      nextPageToken: "page-2",
    })).toEqual({ mode: "cursor", canFirstPage: false, canPreviousPage: false, canNextPage: true })

    expect(getCursorPaginationNavigation({
      currentPage: 2,
      pageTokens: { 1: undefined, 2: "page-3" },
    })).toEqual({ mode: "cursor", canFirstPage: true, canPreviousPage: true, canNextPage: false })
  })

  it("only accepts adjacent cursor transitions", () => {
    expect(getCursorPageTransition({
      currentPage: 1,
      pageTokens: { 1: undefined },
      nextPageToken: "page-2",
      requestedPage: 2,
    })).toEqual({ reachable: true, pageToken: "page-2" })

    expect(getCursorPageTransition({
      currentPage: 2,
      pageTokens: { 1: undefined, 2: "page-3" },
      requestedPage: 1,
    })).toEqual({ reachable: true, pageToken: undefined })

    expect(getCursorPageTransition({
      currentPage: 1,
      pageTokens: { 1: undefined },
      nextPageToken: "page-2",
      requestedPage: 3,
    })).toEqual({ reachable: false })

    expect(getCursorPageTransition({
      currentPage: 0,
      firstPage: 0,
      pageTokens: { 0: undefined },
      nextPageToken: "page-1",
      requestedPage: -1,
    })).toEqual({ reachable: false })

    expect(getCursorPageTransition({
      currentPage: 1,
      pageTokens: { 1: undefined },
      requestedPage: 0,
    })).toEqual({ reachable: false })
  })

  it("resets token and page index when search, filters, or sorting change", () => {
    const query = createBusinessListQuery({
      pageSize: 20,
      pageIndex: 3,
      pageToken: "token-page-3",
      search: "admin",
      filters: { tags: ["fuzz"] },
      sorting: { field: "updatedAt", direction: "desc" },
    })

    const searched = applyBusinessListControlChange(query, { search: "api" })
    expect(searched.pageIndex).toBe(1)
    expect(searched.pageToken).toBeUndefined()

    const filtered = applyBusinessListControlChange(query, { filters: { tags: ["custom"] } })
    expect(filtered.pageIndex).toBe(1)
    expect(filtered.pageToken).toBeUndefined()

    const sorted = applyBusinessListControlChange(query, { sorting: { field: "displayName", direction: "asc" } })
    expect(sorted.pageIndex).toBe(1)
    expect(sorted.pageToken).toBeUndefined()
  })

  it("preserves search, filters, and sorting when only page changes", () => {
    const query: BusinessListQuery = {
      pageSize: 20,
      pageIndex: 1,
      search: "admin",
      filters: { tags: ["fuzz"] },
      sorting: { field: "updatedAt", direction: "desc" },
    }

    expect(setBusinessListPage(query, { pageIndex: 2, pageToken: "next-token" })).toEqual({
      ...query,
      pageIndex: 2,
      pageToken: "next-token",
    })
  })

  it("compiles ordinary search and multi-select facets into grouped filter expressions", () => {
    const filter = compileBusinessListFilter(
      {
        search: "admin \"api\"",
        filters: {
          tags: ["fuzz", "subdomain", "fuzz", ""],
          status: ["active"],
          empty: [],
        },
      },
      {
        search: { field: "displayName", operator: "=" },
        facets: {
          tags: { field: "tags", operator: "==" },
          status: { field: "status", operator: "==" },
        },
      }
    )

    expect(filter).toBe(
      `(tags=="fuzz" || tags=="subdomain") && status=="active" && displayName="admin \\"api\\""`
    )
  })

  it("compiles ordinary search across multiple backend fields as an OR group", () => {
    const filter = compileBusinessListFilter(
      {
        search: "edge-01",
        filters: {
          status: ["online", "offline"],
          healthState: ["healthy"],
        },
      },
      {
        search: [
          { field: "displayName", operator: "=" },
          { field: "observedHostname", operator: "=" },
          { field: "connectionIp", operator: "=" },
        ],
        facets: {
          status: { field: "status", operator: "==" },
          healthState: { field: "healthState", operator: "==" },
        },
      }
    )

    expect(filter).toBe(
      `(status=="online" || status=="offline") && healthState=="healthy" && (displayName="edge-01" || observedHostname="edge-01" || connectionIp="edge-01")`
    )
  })

  it("preserves URL search bytes without presentation trimming", () => {
    const rawURL = "HTTPS://Example.test/path?payload=%00%zz#fragment "
    const filter = compileBusinessListFilter(
      { search: rawURL, filters: {} },
      { search: { field: "url" }, facets: {} },
    )

    expect(filter).toBe(`url="${rawURL.replace(/\\/g, "\\\\").replace(/"/g, '\\\"')}"`)
  })

  it("emits explicit equality syntax when a URL search is configured as exact", () => {
    const filter = compileBusinessListFilter(
      { search: "https://Example.test/path?payload=%00%zz", filters: {} },
      { search: { field: "url", operator: "==" }, facets: {} },
    )

    expect(filter).toBe('url=="https://Example.test/path?payload=%00%zz"')
  })

  it("treats only all-whitespace URL input as absent", () => {
    const rawURL = "HTTPS://Example.test/path?payload=%00%zz#fragment "

    expect(preserveRawURLSearchInput(rawURL)).toBe(rawURL)
    expect(preserveRawURLSearchInput("   ")).toBeUndefined()
  })

  it("compiles orderBy only from declared sortable backend fields", () => {
    const registry = {
      displayName: { orderBy: "displayName", firstDirection: "asc" as const },
      updatedAt: { orderBy: "updatedAt", firstDirection: "desc" as const },
    }

    expect(compileBusinessListOrderBy({ field: "updatedAt", direction: "desc" }, registry)).toBe("updatedAt desc")
    expect(compileBusinessListOrderBy({ field: "displayName", direction: "asc" }, registry)).toBe("displayName")
    expect(compileBusinessListOrderBy({ field: "actions", direction: "asc" }, registry)).toBeUndefined()
    expect(compileBusinessListOrderBy(undefined, registry)).toBeUndefined()
  })

  it("cycles declared sorting through first direction, opposite direction, and endpoint default", () => {
    const registry = {
      displayName: { orderBy: "displayName", firstDirection: "asc" as const },
    }
    const defaultSorting = { field: "updatedAt", direction: "desc" as const }
    const query = createBusinessListQuery({
      pageSize: 20,
      pageIndex: 2,
      pageToken: "token-page-2",
      sorting: defaultSorting,
    })

    const first = toggleBusinessListSorting(query, "displayName", registry, defaultSorting)
    expect(first.sorting).toEqual({ field: "displayName", direction: "asc" })
    expect(first.pageIndex).toBe(1)
    expect(first.pageToken).toBeUndefined()

    const opposite = toggleBusinessListSorting(first, "displayName", registry, defaultSorting)
    expect(opposite.sorting).toEqual({ field: "displayName", direction: "desc" })

    const reset = toggleBusinessListSorting(opposite, "displayName", registry, defaultSorting)
    expect(reset.sorting).toEqual(defaultSorting)

    expect(toggleBusinessListSorting(reset, "actions", registry, defaultSorting)).toBe(reset)
  })
})
