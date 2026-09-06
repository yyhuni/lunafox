import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/screenshots/screenshots-gallery-state.ts"), "utf8")

describe("screenshots-gallery-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useScreenshotsGalleryState")
    expect(source).toContain("from \"react\"")
  })

  it("uses shared business-list query helpers instead of legacy page/filter local state", () => {
    expect(source).toContain("createBusinessListQuery")
    expect(source).toContain("applyBusinessListControlChange")
    expect(source).toContain("setBusinessListPage")
    expect(source).toContain("compileBusinessListFilter")
    expect(source).toContain("compileBusinessListOrderBy")
    expect(source).not.toContain("const [pagination, setPagination]")
    expect(source).not.toContain("const [filterQuery, setFilterQuery]")
  })

  it("declares screenshot query semantics as url search, statusCode filter, and approved sort fields", () => {
    expect(source).toContain("SCREENSHOT_FILTER_CONFIG")
    expect(source).toContain('search: { field: "url", operator: "==" }')
    expect(source).toContain('statusCode: { field: "statusCode" }')
    expect(source).toContain("SCREENSHOT_SORTABLE_FIELDS")
    expect(source).toContain('statusCode: { orderBy: "statusCode"')
    expect(source).toContain('createdAt: { orderBy: "createdAt"')
    expect(source).toContain('SCREENSHOT_DEFAULT_SORTING: BusinessListSorting = { field: "createdAt", direction: "desc" }')
    expect(source).not.toContain('orderBy: "url"')
    expect(source).not.toContain('orderBy: "image"')
    expect(source).not.toContain('orderBy: "updatedAt"')
  })

  it("maintains separate target and scan page-token state and isolates scope changes immediately", () => {
    expect(source).toContain("const [pageTokens, setPageTokens]")
    expect(source).toContain("const [scanPageTokens, setScanPageTokens]")
    expect(source).toContain("useCursorPaginationScopeChange")
    expect(source).toContain("pageToken: targetPageToken")
    expect(source).toContain("pageToken: scanPageToken")
    expect(source).toContain("isPlaceholderData || hasCursorScopeChanged")
  })

  it("uses scoped backend status-code options instead of current-page row options", () => {
    expect(source).toContain("useTargetScreenshotFilterOptions")
    expect(source).toContain("useScanScreenshotFilterOptions")
    expect(source).toContain("useScopedScreenshotFilterOptions")
    expect(source).toContain('useScopedScreenshotFilterOptions("statusCode"')
    expect(source).toContain("normalizeFilterOptions")
    expect(source).not.toContain("buildCurrentPageOptions")
    expect(source).not.toContain("screenshot.statusCode")
  })

  it("keeps select-all and clear-selection commands separate for the shared action bar", () => {
    expect(source).toContain("const selectAll = React.useCallback")
    expect(source).toContain("const clearSelection = React.useCallback")
    expect(source).toContain("clearSelection,")
    expect(source).not.toContain("selectedIds.size === screenshots.length")
  })
})
