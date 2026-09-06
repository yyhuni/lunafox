import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/endpoints/endpoints-detail-view-state.ts"), "utf8")
const filterConfigSource = source.slice(
  source.indexOf("const ENDPOINT_FILTER_CONFIG"),
  source.indexOf("function uniqueNonEmptyValues")
)

describe("endpoints-detail-view-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useEndpointsDetailViewState")
    expect(source).toContain("from \"react\"")
  })

  it("uses BusinessListQuery as the single endpoint list query state", () => {
    expect(source).toContain("createBusinessListQuery")
    expect(source).toContain("applyBusinessListControlChange")
    expect(source).toContain("setBusinessListPage")
    expect(source).toContain("toggleBusinessListSorting")
    expect(source).not.toContain("useSearchState")
    expect(source).not.toContain("filterQuery, setFilterQuery")
  })

  it("compiles endpoint search and facets to canonical lowerCamelCase filters", () => {
    expect(source).toContain("ENDPOINT_FILTER_CONFIG")
    expect(filterConfigSource).toContain('search: { field: "url", operator: "==" }')
    expect(filterConfigSource).toContain('statusCode: { field: "statusCode" }')
    expect(filterConfigSource).toContain('tech: { field: "tech" }')
    expect(filterConfigSource).toContain('webserver: { field: "webserver" }')
    expect(filterConfigSource).toContain('contentType: { field: "contentType" }')
    expect(filterConfigSource).toContain('vhost: { field: "vhost" }')
    expect(filterConfigSource).not.toContain("status_code")
    expect(filterConfigSource).not.toContain("content_type")
  })

  it("compiles endpoint orderBy and preserves target/scan page token maps independently", () => {
    expect(source).toContain("ENDPOINT_SORTABLE_FIELDS")
    expect(source).not.toContain('url: { orderBy: "url"')
    expect(source).toContain('statusCode: { orderBy: "statusCode"')
    expect(source).toContain('contentLength: { orderBy: "contentLength"')
    expect(source).toContain('createdAt: { orderBy: "createdAt"')
    expect(source).toContain("ENDPOINT_DEFAULT_SORTING")
    expect(source).toContain("pageTokens")
    expect(source).toContain("scanPageTokens")
    expect(source).toContain("nextPageToken")
    expect(source).toContain("useCursorPaginationScopeChange")
    expect(source).toContain("pageToken: targetPageToken")
    expect(source).toContain("pageToken: scanPageToken")
    expect(source).toContain("isPlaceholderData || hasCursorScopeChanged")
  })

  it("uses parent-scoped endpoint filter options instead of current-page option derivation", () => {
    expect(source).toContain("useTargetEndpointFilterOptions")
    expect(source).toContain("useScanEndpointFilterOptions")
    expect(source).toContain("normalizeFilterOptions")
    expect(source).not.toContain("data?.results.map")
    expect(source).not.toContain("data?.results.flatMap")
  })

  it("defaults secondary location, host, and response payload columns to hidden while keeping visibility user-controlled", () => {
    expect(source).toContain("DEFAULT_ENDPOINT_COLUMN_VISIBILITY")
    expect(source).toContain("host: false")
    expect(source).toContain("location: false")
    expect(source).toContain("responseBody: false")
    expect(source).toContain("responseHeaders: false")
    expect(source).toContain("columnVisibility")
    expect(source).toContain("setColumnVisibility")
  })
})
