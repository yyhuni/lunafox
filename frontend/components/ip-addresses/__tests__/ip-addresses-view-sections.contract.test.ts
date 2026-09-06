import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ip-addresses/ip-addresses-view-sections.tsx"), "utf8")
const viewSource = readFileSync(path.resolve(process.cwd(), "components/ip-addresses/ip-addresses-view.tsx"), "utf8")

describe("ip-addresses-view-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).not.toContain("IPAddressesViewErrorState")
    expect(viewSource).toContain("AppErrorState")
    expect(viewSource).toContain('notFoundKind: "unexpected-error"')
    expect(viewSource).toContain('variant="section"')
    expect(source).toContain("export function IPAddressesViewRouteFallback")
  })

  it("keeps the table loading state aligned to the route gutter", () => {
    expect(source).toContain("IPAddressesViewLoadingState")
    expect(source).toContain("state: IPAddressesViewState")
    expect(source).toContain("<IPAddressesDataTable")
    expect(source).toContain("loading")
    expect(source).toContain("initialLoading")
    expect(source).toContain("loadingRowCount={rowCount}")
    expect(source).toContain("data={[]}")
    expect(source).not.toContain('from "./ip-addresses-view-skeleton"')
    expect(source).not.toContain("return <IPAddressesViewSkeleton rowCount={rowCount} />")
  })

  it("renders the state-less route fallback through the real IP table loading mode", () => {
    expect(source).toContain('import { createIPAddressColumns } from "./ip-addresses-columns"')
    expect(source).toContain("const columns = createIPAddressColumns")
    expect(source).toContain("pagination={{ pageIndex: 0, pageSize: IP_ADDRESSES_ROUTE_FALLBACK_PAGE_SIZE }}")
    expect(source).toContain("portFilter={[]}")
    expect(source).toContain("sortingMode=\"server\"")
    expect(source).toContain("loadingRowCount={rowCount}")
    expect(source).not.toMatch(/\bDataTableSkeleton\b/)
    expect(source).not.toContain("IP_ADDRESSES_ROUTE_FALLBACK_COLUMN_COUNT")
    expect(source).not.toContain("IP_ADDRESSES_ROUTE_FALLBACK_TOOLBAR_BUTTON_COUNT")
  })

  it("keeps the state-less route fallback on the resolved page gutter", () => {
    const fallbackStart = source.indexOf("export function IPAddressesViewRouteFallback")
    const contentStart = source.indexOf("export function IPAddressesViewContent")
    const fallbackSource = source.slice(fallbackStart, contentStart)

    expect(source).toContain('import { DetailAssetContentFrame }')
    expect(fallbackSource).toContain("<DetailAssetContentFrame>")
    expect(fallbackSource).toContain("</DetailAssetContentFrame>")
  })

  it("keeps the ordinary loading state inside the page-owned frame", () => {
    const loadingStart = source.indexOf("export function IPAddressesViewLoadingState")
    const fallbackStart = source.indexOf("export function IPAddressesViewRouteFallback")
    const loadingSource = source.slice(loadingStart, fallbackStart)

    expect(loadingSource).not.toContain("DetailAssetContentFrame")
  })
})
