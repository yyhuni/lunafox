import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/endpoints/endpoints-detail-view.tsx"), "utf8")

describe("endpoints-detail-view contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function EndpointsDetailView")
    expect(source).toContain("from \"./endpoints-detail-view-sections\"")
  })

  it("uses shared skeleton handoff around both content and dialogs", () => {
    expect(source).toContain("ContentHandoff")
    expect(source).toContain("const isInitialLoading = state.isLoading && !state.data")
    expect(source).toContain("const loadingRowCount = getDataTableSkeletonRowCount(state.pagination.pageSize)")
    expect(source).toContain('owner="endpoints-detail-view-content"')
    expect(source).toContain("skeleton={<EndpointsDetailViewLoadingState state={state} rowCount={loadingRowCount} />}")
    expect(source).toContain("<EndpointsDetailViewContent state={state} onRowClick={handleSelectEndpoint} />")
    expect(source).toContain("<EndpointsDetailViewDialogs state={state} />")
    expect(source).not.toContain("ContentReveal")
  })

  it("keeps endpoint inspection selection in view-local drawer state", () => {
    expect(source).toContain("EndpointDetailDrawer")
    expect(source).toContain("activeEndpoint")
    expect(source).toContain("handleSelectEndpoint")
    expect(source).toContain("handleEndpointDetailOpenChange")
  })
})
