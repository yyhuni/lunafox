import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/directories/directories-view-state.ts"), "utf8")

describe("directories-view-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useDirectoriesViewState")
    expect(source).toContain("useDirectoryTableColumns")
    expect(source).toContain("from \"react\"")
  })

  it("does not resolve translations for directory columns that are not rendered", () => {
    expect(source).not.toContain('tColumns("directory.words")')
    expect(source).not.toContain('tColumns("directory.lines")')
    expect(source).not.toContain('tColumns("directory.duration")')
  })

  it("delegates selected-row export to the exact Directory CSV formatter", () => {
    expect(source).toContain('import { buildDirectoryCSV } from "./directory-csv"')
    expect(source).toContain("buildDirectoryCSV(selectedDirectories)")
    expect(source).not.toContain("formatDurationNsToMs")
    expect(source).not.toContain("item.words")
    expect(source).not.toContain("item.lines")
    expect(source).not.toContain("item.websiteUrl")
  })

  it("uses the shared business-list query helpers instead of legacy local search state", () => {
    expect(source).toContain("createBusinessListQuery")
    expect(source).toContain("applyBusinessListControlChange")
    expect(source).toContain("setBusinessListPage")
    expect(source).toContain("toggleBusinessListSorting")
    expect(source).toContain("compileBusinessListFilter")
    expect(source).toContain("compileBusinessListOrderBy")
    expect(source).not.toContain("useSearchState")
    expect(source).not.toContain("filterQuery, setFilterQuery")
  })

  it("declares the directory query contract as url search, status/contentType filters, and approved sort fields", () => {
    expect(source).toContain("DIRECTORY_FILTER_CONFIG")
    expect(source).toContain('search: { field: "url", operator: "==" }')
    expect(source).toContain('status: { field: "status" }')
    expect(source).toContain('contentType: { field: "contentType" }')
    expect(source).toContain("DIRECTORY_SORTABLE_FIELDS")
    expect(source).toContain('status: { orderBy: "status"')
    expect(source).toContain('contentLength: { orderBy: "contentLength"')
    expect(source).toContain('createdAt: { orderBy: "createdAt"')
    expect(source).toContain('DIRECTORY_DEFAULT_SORTING: BusinessListSorting = { field: "createdAt", direction: "desc" }')
    expect(source).not.toContain('orderBy: "url"')
    expect(source).not.toContain('orderBy: "contentType"')
  })

  it("maintains separate target and scan page-token state and isolates scope changes immediately", () => {
    expect(source).toContain("const [pageTokens, setPageTokens]")
    expect(source).toContain("const [scanPageTokens, setScanPageTokens]")
    expect(source).toContain("useCursorPaginationScopeChange")
    expect(source).toContain('`target:${targetId}\\u0000${websiteScopeKey}`')
    expect(source).toContain("pageToken: targetPageToken")
    expect(source).toContain("pageToken: scanPageToken")
    expect(source).toContain("isPlaceholderData || hasCursorScopeChanged")
  })

  it("uses scoped backend filter options instead of current-page row options", () => {
    expect(source).toContain("useTargetDirectoryFilterOptions")
    expect(source).toContain("useScanDirectoryFilterOptions")
    expect(source).toContain("useScopedDirectoryFilterOptions")
    expect(source).toContain('useScopedDirectoryFilterOptions("status"')
    expect(source).toContain('useScopedDirectoryFilterOptions("contentType"')
    expect(source).toContain("normalizeFilterOptions")
    expect(source).not.toContain("buildCurrentPageOptions")
    expect(source).not.toContain("directory.status")
    expect(source).not.toContain("directory.contentType")
  })
})
