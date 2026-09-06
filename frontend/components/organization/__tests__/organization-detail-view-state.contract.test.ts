import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/organization-detail-view-state.ts"), "utf8")

describe("organization-detail-view-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useOrganizationDetailViewState")
    expect(source).toContain("useScheduledScans")
    expect(source).toContain("organizationId: organizationNumericId")
    expect(source).toContain("createScheduledScanColumns")
    expect(source).toContain("const isInitialContentLoading = isLoading || isLoadingSummaryTargets || isLoadingScheduledScans")
    expect(source).toContain("isInitialContentLoading,")
    expect(source).toContain("from \"react\"")
  })

  it("stores organization target type filtering as a multi-select value", () => {
    expect(source).toContain("export type OrganizationTargetTypeFilter = TargetType[]")
    expect(source).toContain("const typeFilter = (targetQuery.filters.type ?? []) as OrganizationTargetTypeFilter")
    expect(source).toContain("const handleTypeFilterChange = React.useCallback((value: OrganizationTargetTypeFilter)")
    expect(source).not.toContain('React.useState<TargetType | "">("")')
  })

  it("combines multi-selected target types into smart filter syntax while preserving single-type API filtering", () => {
    expect(source).toContain("function buildTargetTypeFilterQuery")
    expect(source).toContain('`type="${type}"`')
    expect(source).toContain('.join(" || ")')
    expect(source).toContain("function combineTargetFilterQuery")
    expect(source).toContain("const typeParam = typeFilter.length === 1 ? typeFilter[0] : undefined")
    expect(source).toContain("const targetFilterParam = combineTargetFilterQuery(searchQuery, typeFilter)")
    expect(source).toContain("search: typeFilter.length <= 1 ? searchQuery || undefined : undefined")
    expect(source).toContain("filter: targetFilterParam")
    expect(source).toContain("type: typeParam")
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

  it("stores scheduled status and workflow filters as multi-value facets", () => {
    expect(source).toContain('export type OrganizationScheduledStatusFilter = Array<"enabled" | "paused">')
    expect(source).toContain("export type OrganizationScheduledWorkflowFilter = string[]")
    expect(source).toContain("React.useState<OrganizationScheduledStatusFilter>([])")
    expect(source).toContain("React.useState<OrganizationScheduledWorkflowFilter>([])")
    expect(source).toContain('scheduledStatusFilter.includes("enabled")')
    expect(source).toContain('scheduledStatusFilter.includes("paused")')
    expect(source).toContain("scheduledWorkflowFilter.includes(scan.scanWorkflow)")
    expect(source).not.toContain('scheduledWorkflowFilter === "all"')
  })
})
