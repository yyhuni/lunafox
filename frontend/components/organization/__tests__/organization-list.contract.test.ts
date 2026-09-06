import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/organization-list.tsx"), "utf8")

describe("organization-list contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function OrganizationList")
    expect(source).toContain("from \"./organization-list-sections\"")
  })

  it("uses shared skeleton handoff instead of hard-cutting from loading to content", () => {
    expect(source).toContain("ContentHandoff")
    expect(source).toContain("const isInitialLoading = state.isLoading || !state.data")
    expect(source).toContain("const loadingRowCount = getDataTableSkeletonRowCount(state.pagination.pageSize)")
    expect(source).toContain('owner="organization-list-content"')
    expect(source).toContain("skeleton={<OrganizationListLoadingState state={state} rowCount={loadingRowCount} />}")
    expect(source).toContain("prepareContentBeforeHandoff")
    expect(source).not.toContain('className="space-y-4"')
    expect(source).toContain("<OrganizationListTable state={state} stableSurfaceRowCount={loadingRowCount} />")
    expect(source).toContain("<OrganizationListDialogs state={state} />")
    expect(source).not.toContain("OrganizationListSkeleton")
    expect(source).not.toContain('skeletonClassName="flex min-h-0 flex-1 flex-col"')
  })
})
