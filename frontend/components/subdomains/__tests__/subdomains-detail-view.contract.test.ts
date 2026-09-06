import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/subdomains/subdomains-detail-view.tsx"), "utf8")

describe("subdomains-detail-view contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function SubdomainsDetailView")
    expect(source).toContain("from \"./subdomains-detail-view-sections\"")
  })

  it("uses shared skeleton handoff around both content and dialogs", () => {
    expect(source).toContain("ContentHandoff")
    expect(source).toContain("const isInitialLoading = state.isLoading && !state.subdomainsData")
    expect(source).toContain("const loadingRowCount = getDataTableSkeletonRowCount(state.pagination.pageSize)")
    expect(source).toContain('owner="subdomains-detail-view-content"')
    expect(source).toContain("skeleton={<SubdomainsDetailViewLoadingState state={state} rowCount={loadingRowCount} />}")
    expect(source).toContain("<SubdomainsDetailViewContent state={state} />")
    expect(source).toContain("<SubdomainsDetailViewDialogs state={state} />")
    expect(source).not.toContain("ContentReveal")
  })
})
