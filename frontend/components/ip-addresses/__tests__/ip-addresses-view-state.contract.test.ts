import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ip-addresses/ip-addresses-view-state.ts"), "utf8")

describe("ip-addresses-view-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useIPAddressesViewState")
    expect(source).toContain("from \"react\"")
  })

  it("uses BusinessListQuery for default search, port filters, sorting, and page tokens", () => {
    expect(source).toContain("createBusinessListQuery")
    expect(source).toContain("applyBusinessListControlChange")
    expect(source).toContain("setBusinessListPage")
    expect(source).toContain("compileBusinessListOrderBy")
    expect(source).toContain("toggleBusinessListSorting")
    expect(source).toContain("const [portFilter, setPortFilter]")
    expect(source).toContain("handlePortFilterChange")
    expect(source).toContain("(ip=\"")
    expect(source).toContain(" || host=\"")
    expect(source).toContain("port")
    expect(source).toContain("pageToken")
    expect(source).toContain("orderBy")
    expect(source).toContain("useCursorPaginationScopeChange")
    expect(source).toContain("pageToken: targetPageToken")
    expect(source).toContain("pageToken: scanPageToken")
    expect(source).toContain("isPlaceholderData || hasCursorScopeChanged")
    expect(source).not.toContain("const selectedPort = portFilter[0]")
    expect(source).not.toContain("page: pagination.pageIndex + 1")
  })
})
