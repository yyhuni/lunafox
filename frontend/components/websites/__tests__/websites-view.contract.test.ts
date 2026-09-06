import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/websites/websites-view.tsx"), "utf8")

describe("websites-view contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function WebSitesView")
    expect(source).toContain("from \"./websites-view-sections\"")
  })

  it("uses shared skeleton handoff instead of hard-cutting from loading to content", () => {
    expect(source).toContain("ContentHandoff")
    expect(source).toContain("const isInitialLoading = state.isLoading && !state.data")
    expect(source).toContain("const loadingRowCount = getDataTableSkeletonRowCount(state.pagination.pageSize)")
    expect(source).toContain('owner="websites-detail-view-content"')
    expect(source).toContain("skeleton={<WebSitesViewLoadingState state={state} rowCount={loadingRowCount} />}")
    expect(source).toContain('from "@/components/shared/feedback/app-error-state"')
    expect(source).toContain("<AppErrorState")
    expect(source).not.toContain("ContentReveal")
  })

  it("keeps the selected website in view-local drawer state", () => {
    expect(source).toContain("WebsiteDetailDrawer")
    expect(source).toContain("activeWebsite")
    expect(source).toContain("handleSelectWebsite")
    expect(source).toContain("handleWebsiteDetailOpenChange")
  })
})
