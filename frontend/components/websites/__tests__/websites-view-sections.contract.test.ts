import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/websites/websites-view-sections.tsx"), "utf8")

describe("websites-view-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function WebSitesViewLoadingState")
    expect(source).toContain("export function WebSitesViewRouteFallback")
    expect(source).toContain("export function WebSitesViewContent")
    expect(source).not.toContain("WebSitesViewErrorState")
  })

  it("passes the shared response-payload visibility state to both table owners", () => {
    expect(source.match(/columnVisibility=\{state\.columnVisibility\}/g)).toHaveLength(2)
    expect(source.match(/onColumnVisibilityChange=\{state\.setColumnVisibility\}/g)).toHaveLength(2)
  })

  it("keeps the table loading state aligned to the route gutter", () => {
    const loadingStart = source.indexOf("export function WebSitesViewLoadingState")
    const fallbackStart = source.indexOf("export function WebSitesViewRouteFallback")
    const loadingSource = source.slice(loadingStart, fallbackStart)
    const contentStart = source.indexOf("export function WebSitesViewContent")
    const fallbackSource = source.slice(fallbackStart, contentStart)

    expect(source).toContain("WebSitesViewLoadingState")
    expect(source).toContain("state: WebSitesViewState")
    expect(source).toContain("<WebSitesDataTable")
    expect(source).toContain("loading")
    expect(source).toContain("loadingRowCount={rowCount}")
    expect(source).toContain("useWebSiteTableColumns")
    expect(source).toContain("DEFAULT_WEBSITE_COLUMN_VISIBILITY")
    expect(source).toContain("export function WebSitesViewRouteFallback({")
    expect(source).toContain("<WebSitesDataTable")
    expect(source).toContain("onBulkAdd={showBulkAdd ? noop : undefined}")
    expect(source).toContain("totalSize: number")
    expect(source).toContain('paginationNavigation={{ mode: "cursor", canFirstPage: false, canPreviousPage: false, canNextPage: false }}')
    expect(source).toContain("cursorPaginationSummary={{ total: totalSize }}")
    expect(source).toContain('import { DetailAssetContentFrame }')
    expect(loadingSource).not.toContain("DetailAssetContentFrame")
    expect(fallbackSource).toContain("<DetailAssetContentFrame>")
    expect(fallbackSource).toContain("</DetailAssetContentFrame>")
    expect(source).toContain("initialLoading")
    expect(source).not.toContain("DataTableSkeleton")
    expect(source).not.toContain("WEBSITES_ROUTE_FALLBACK_COLUMN_COUNT")
    expect(source).not.toContain("WEBSITES_ROUTE_FALLBACK_TOOLBAR_BUTTON_COUNT")
    expect(source).not.toContain('from "./websites-view-skeleton"')
    expect(source).not.toContain("return <WebSitesViewSkeleton rowCount={rowCount} />")
  })
})
