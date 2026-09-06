import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/targets/targets-detail-view-state.ts"), "utf8")

describe("targets-detail-view-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useTargetsDetailViewState")
    expect(source).toContain("from \"react\"")
  })

  it("uses only backend-authorized cursor transitions for organization targets", () => {
    expect(source).toContain("createBusinessListQuery")
    expect(source).toContain("targetPageTokens")
    expect(source).toContain("getCurrentCursorNextPageToken")
    expect(source).toContain("getCursorPaginationNavigation")
    expect(source).toContain("getCursorPageTransition")
    expect(source).toContain("pageToken: targetQuery.pageToken")
    expect(source).toContain("cursorPaginationSummary")
    expect(source).toContain("paginationNavigation")
    expect(source).not.toContain("page: pagination.pageIndex + 1")
  })
})
