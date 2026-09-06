import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/directories/directories-view-sections.tsx"), "utf8")
const viewSource = readFileSync(path.resolve(process.cwd(), "components/directories/directories-view.tsx"), "utf8")

describe("directories-view-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).not.toContain("DirectoriesViewErrorState")
    expect(viewSource).toContain("AppErrorState")
    expect(viewSource).toContain('notFoundKind: "unexpected-error"')
    expect(viewSource).toContain('variant="section"')
    expect(source).toContain("export function DirectoriesViewRouteFallback")
    expect(source).toContain("className")
  })

  it("keeps the table loading state aligned to the route gutter", () => {
    expect(source).toContain("DirectoriesViewLoadingState")
    expect(source).toContain("state: DirectoriesViewState")
    expect(source).toContain("<DirectoriesDataTable")
    expect(source).toContain("loading")
    expect(source).toContain("initialLoading")
    expect(source).toContain("loadingRowCount={rowCount}")
    expect(source).toContain("useDirectoryTableColumns")
    expect(source).toContain("export function DirectoriesViewRouteFallback({")
    expect(source).toContain("<DirectoriesDataTable")
    expect(source).toContain('paginationNavigation={{ mode: "cursor", canFirstPage: false, canPreviousPage: false, canNextPage: false }}')
    expect(source).toContain("cursorPaginationSummary={{ total: totalSize }}")
    expect(source).toContain('<div className="px-4 lg:px-6">')
    expect(source).not.toContain("DataTableSkeleton")
    expect(source).not.toContain("DIRECTORIES_ROUTE_FALLBACK_COLUMN_COUNT")
    expect(source).not.toContain("DIRECTORIES_ROUTE_FALLBACK_TOOLBAR_BUTTON_COUNT")
    expect(source).not.toContain('from "./directories-view-skeleton"')
    expect(source).not.toContain("return <DirectoriesViewSkeleton rowCount={rowCount} />")
  })
})
