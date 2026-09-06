import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/subdomains/subdomains-detail-view-sections.tsx"), "utf8")
const viewSource = readFileSync(path.resolve(process.cwd(), "components/subdomains/subdomains-detail-view.tsx"), "utf8")

describe("subdomains-detail-view-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).not.toContain("SubdomainsDetailViewErrorState")
    expect(viewSource).toContain("AppErrorState")
    expect(viewSource).toContain('notFoundKind: "unexpected-error"')
    expect(viewSource).toContain('variant="section"')
    expect(source).toContain("export function SubdomainsDetailViewRouteFallback")
  })

  it("keeps the table loading state aligned to the route gutter", () => {
    const loadingStart = source.indexOf("export function SubdomainsDetailViewLoadingState")
    const fallbackStart = source.indexOf("export function SubdomainsDetailViewRouteFallback")
    const loadingSource = source.slice(loadingStart, fallbackStart)

    expect(source).toContain("SubdomainsDetailViewLoadingState")
    expect(source).toContain("state: SubdomainsDetailViewState")
    expect(source).toContain("<SubdomainsDataTable")
    expect(source).toContain("loading")
    expect(source).toContain("initialLoading")
    expect(source).toContain("loadingRowCount={rowCount}")
    expect(source).toContain("data={[]}")
    expect(source).not.toContain('from "./subdomains-detail-view-skeleton"')
    expect(source).not.toContain("return <SubdomainsDetailViewSkeleton rowCount={rowCount} />")
    expect(loadingSource).not.toContain("DetailAssetContentFrame")
  })

  it("renders the state-less route fallback through the real subdomains table loading mode", () => {
    const fallbackStart = source.indexOf("export function SubdomainsDetailViewRouteFallback")
    const contentStart = source.indexOf("export function SubdomainsDetailViewContent")
    const fallbackSource = source.slice(fallbackStart, contentStart)

    expect(source).toContain('import { createSubdomainColumns } from "./subdomains-columns"')
    expect(source).toContain("const columns = createSubdomainColumns")
    expect(source).toContain("pagination={{ pageIndex: 0, pageSize: SUBDOMAINS_ROUTE_FALLBACK_PAGE_SIZE }}")
    expect(source).toContain("sortingMode=\"server\"")
    expect(source).toContain("loadingRowCount={rowCount}")
    expect(source).toContain('import { DetailAssetContentFrame }')
    expect(fallbackSource).toContain("<DetailAssetContentFrame>")
    expect(fallbackSource).toContain("</DetailAssetContentFrame>")
    expect(source).not.toMatch(/\bDataTableSkeleton\b/)
    expect(source).not.toContain("SUBDOMAINS_ROUTE_FALLBACK_COLUMN_COUNT")
    expect(source).not.toContain("SUBDOMAINS_ROUTE_FALLBACK_TOOLBAR_BUTTON_COUNT")
  })
})
