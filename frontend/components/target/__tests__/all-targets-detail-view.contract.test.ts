import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/target/all-targets-detail-view.tsx"), "utf8")
const guide = readFileSync(path.resolve(process.cwd(), "components/target/README.md"), "utf8")

describe("all-targets-detail-view contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function AllTargetsDetailView")
    expect(source).toContain("from \"./all-targets-detail-view-sections\"")
  })

  it("uses shared skeleton handoff instead of a hard initial loading branch", () => {
    expect(source).toContain("ContentHandoff")
    expect(source).toContain("const isInitialLoading = state.isLoading && !state.data")
    expect(source).toContain("const loadingRowCount = getDataTableSkeletonRowCount(state.pagination.pageSize)")
    expect(source).toContain("if (!isInitialLoading && state.error)")
    expect(source).toContain('owner="all-targets-detail-view-content"')
    expect(source).toContain("skeleton={<AllTargetsDetailViewLoadingState state={state} rowCount={loadingRowCount} />}")
    expect(source).toContain("<AllTargetsDetailViewTable state={state} stableSurfaceRowCount={loadingRowCount} />")
    expect(source).toContain("<AllTargetsDetailViewDialogs state={state} />")
    expect(source).not.toContain("ContentReveal")
  })

  it("keeps the target list handoff free of a workspace-height override", () => {
    expect(source).toContain('layer="workspace"')
    expect(source).not.toContain('className="flex min-h-0 flex-1 flex-col"')
    expect(source).not.toContain('skeletonClassName="flex min-h-0 flex-1 flex-col"')
  })

  it("documents the target list as a header-plus-workspace handoff pattern", () => {
    expect(guide).toContain("## Target List Workspace Handoff")
    expect(guide).toContain("`AllTargetsDetailView` own the first-screen workspace wait")
    expect(guide).toContain("`header -> workspace skeleton -> workspace content`")
    expect(guide).toContain("Do not add a route-level")
  })
})
