import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/vulnerabilities/vulnerabilities-detail-view-state.ts"), "utf8")

describe("vulnerabilities-detail-view-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useVulnerabilitiesDetailViewState")
    expect(source).toContain("from \"react\"")
  })

  it("keeps the list state independent from row detail activation", () => {
    expect(source).not.toContain("onViewDetail")
    expect(source).not.toContain("useInteractionOpenLoader")
    expect(source).not.toContain("detail-dialog")
  })

  it("uses BusinessListQuery for canonical search, filters, sorting, and page tokens", () => {
    expect(source).toContain("createBusinessListQuery")
    expect(source).toContain("compileBusinessListFilter")
    expect(source).toContain("compileBusinessListOrderBy")
    expect(source).toContain("setBusinessListPage")
    expect(source).toContain("toggleBusinessListSorting")
    expect(source).toContain("VULNERABILITY_DEFAULT_SORTING")
    expect(source).toContain('createdAt: { orderBy: "createdAt", firstDirection: "desc" }')
    expect(source).toContain('search: { field: "url", operator: "==" }')
    expect(source).toContain('severity: { field: "severity", operator: "==" }')
    expect(source).toContain('source: { field: "source", operator: "==" }')
    expect(source).toContain('vulnType: { field: "vulnType", operator: "==" }')
    expect(source).toContain('isReviewed: { field: "isReviewed", operator: "==" }')
    expect(source).toContain("const [pageTokens, setPageTokens]")
    expect(source).toContain("const [targetPageTokens, setTargetPageTokens]")
    expect(source).toContain("const [scanPageTokens, setScanPageTokens]")
    expect(source).toContain("useCursorPaginationScopeChange")
    expect(source).toContain("const activePageToken = hasCursorScopeChanged ? undefined : activeBusinessQuery.pageToken")
    expect(source).toContain("isPlaceholderData || hasCursorScopeChanged")
    expect(source).not.toContain("function combineFilterQuery(")
    expect(source).not.toContain("severityParam")
    expect(source).not.toContain("isReviewedParam")
  })

  it("wires finite severity facets locally and dynamic source/type facets to backend options", () => {
    expect(source).toContain("const [severityFilter, setSeverityFilter] = React.useState<SeverityFilter>([])")
    expect(source).toContain("const [sourceFilter, setSourceFilter] = React.useState<string[]>([])")
    expect(source).toContain("const [vulnTypeFilter, setVulnTypeFilter] = React.useState<string[]>([])")
    expect(source).toContain("useScopedVulnerabilityFilterOptions")
    expect(source).toContain('useScopedVulnerabilityFilterOptions("source"')
    expect(source).toContain('useScopedVulnerabilityFilterOptions("vulnType"')
    expect(source).toContain("normalizeFilterOptions")
    expect(source).not.toContain('React.useState<SeverityFilter>("all")')
  })

  it("keeps review tabs scoped away from scan vulnerabilities", () => {
    expect(source).toContain("supportsReviewTabs")
    expect(source).toContain("!scanId")
    expect(source).toContain("reviewFilters")
    expect(source).toContain('isReviewed: reviewFilter === "reviewed" ? ["true"]')
    expect(source).toContain('reviewFilter === "pending" ? ["false"]')
    expect(source).toContain("onReviewFilterChange: supportsReviewTabs ? handleReviewFilterChange : undefined")
  })
})
