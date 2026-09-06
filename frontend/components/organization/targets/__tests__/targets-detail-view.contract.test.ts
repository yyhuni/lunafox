import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/targets/targets-detail-view.tsx"), "utf8")

describe("targets-detail-view contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function OrganizationTargetsDetailView")
    expect(source).toContain("from \"./targets-detail-view-sections\"")
  })

  it("keeps empty and error states from flashing before the initial handoff completes", () => {
    expect(source).toContain("ContentHandoff")
    expect(source).toContain("const isInitialLoading = state.isLoading && !state.organization")
    expect(source).toContain("if (!isInitialLoading && state.error)")
    expect(source).toContain("if (!isInitialLoading && !state.organization)")
    expect(source).toContain('owner="organization-targets-detail-view-content"')
    expect(source).toContain("skeleton={<TargetsDetailViewLoadingState state={state} />}")
    expect(source).not.toContain("getDataTableSkeletonRowCount")
    expect(source).toContain("{state.organization ? (")
    expect(source).not.toContain("ContentReveal")
  })
})
