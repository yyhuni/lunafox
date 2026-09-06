import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/endpoints/endpoints-detail-view-sections.tsx"), "utf8")
const viewSource = readFileSync(path.resolve(process.cwd(), "components/endpoints/endpoints-detail-view.tsx"), "utf8")

describe("endpoints-detail-view-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).not.toContain("EndpointsDetailViewErrorState")
    expect(viewSource).toContain("AppErrorState")
    expect(viewSource).toContain('notFoundKind: "unexpected-error"')
    expect(viewSource).toContain('variant="section"')
    expect(source).toContain("export function EndpointsDetailViewRouteFallback")
    expect(source).toContain('import { DetailAssetContentFrame }')
  })

  it("routes initial loading through the real endpoint data table owner", () => {
    const loadingStart = source.indexOf("export function EndpointsDetailViewLoadingState")
    const fallbackStart = source.indexOf("export function EndpointsDetailViewRouteFallback")
    const loadingSource = source.slice(loadingStart, fallbackStart)

    expect(source).toContain("EndpointsDetailViewLoadingState")
    expect(source).toContain("state: EndpointsDetailViewState")
    expect(source).toContain("<EndpointsDataTable")
    expect(source).toContain("loading")
    expect(source).toContain("initialLoading")
    expect(source).toContain("loadingRowCount={rowCount}")
    expect(source).toContain("data={[]}")
    expect(source).not.toContain("EndpointsDetailViewSkeleton")
    expect(source).not.toContain("return <EndpointsDetailViewSkeleton rowCount={rowCount} />")
    expect(source).not.toContain("DataTableSkeleton owner=\"endpoints-detail-view\"")
    expect(loadingSource).not.toContain("DetailAssetContentFrame")
  })

  it("passes endpoint column visibility through loading and resolved table owners", () => {
    expect(source).toContain("columnVisibility={state.columnVisibility}")
    expect(source).toContain("onColumnVisibilityChange={state.setColumnVisibility}")
  })

  it("renders the state-less route fallback through the real endpoint table loading mode", () => {
    const fallbackStart = source.indexOf("export function EndpointsDetailViewRouteFallback")
    const contentStart = source.indexOf("export function EndpointsDetailViewContent")
    const fallbackSource = source.slice(fallbackStart, contentStart)

    expect(source).toContain('import { createEndpointColumns } from "./endpoints-columns"')
    expect(source).toContain("const columns = createEndpointColumns")
    expect(source).toContain("pagination={{ pageIndex: 0, pageSize: ENDPOINTS_ROUTE_FALLBACK_PAGE_SIZE }}")
    expect(source).toContain('import { DEFAULT_ENDPOINT_COLUMN_VISIBILITY } from "./endpoints-detail-view-state"')
    expect(source).toContain("columnVisibility={DEFAULT_ENDPOINT_COLUMN_VISIBILITY}")
    expect(source).toContain('import { DetailAssetContentFrame }')
    expect(fallbackSource).toContain("<DetailAssetContentFrame>")
    expect(fallbackSource).toContain("</DetailAssetContentFrame>")
    expect(source).toContain("sortingMode=\"server\"")
    expect(source).toContain("loadingRowCount={rowCount}")
    expect(source).not.toMatch(/\bDataTableSkeleton\b/)
    expect(source).not.toContain("ENDPOINTS_ROUTE_FALLBACK_COLUMN_COUNT")
    expect(source).not.toContain("ENDPOINTS_ROUTE_FALLBACK_TOOLBAR_BUTTON_COUNT")
  })
})
