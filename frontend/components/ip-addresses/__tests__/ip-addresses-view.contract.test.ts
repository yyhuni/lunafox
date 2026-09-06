import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ip-addresses/ip-addresses-view.tsx"), "utf8")

describe("ip-addresses-view contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function IPAddressesView")
    expect(source).toContain("from \"./ip-addresses-view-sections\"")
  })

  it("uses shared skeleton handoff instead of hard-cutting from loading to content", () => {
    expect(source).toContain("ContentHandoff")
    expect(source).toContain("const isInitialLoading = state.isLoading && !state.data")
    expect(source).toContain("const loadingRowCount = getDataTableSkeletonRowCount(state.pagination.pageSize)")
    expect(source).toContain('owner="ip-addresses-detail-view-content"')
    expect(source).toContain("skeleton={<IPAddressesViewLoadingState state={state} rowCount={loadingRowCount} />}")
    expect(source).not.toContain("ContentReveal")
  })
})
