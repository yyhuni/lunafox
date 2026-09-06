import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-history-list-view-state.ts"), "utf8")

describe("scan-history-list-view-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useScanHistoryListViewState")
    expect(source).toContain("from \"react\"")
  })

  it("wires runtime details to the row instead of the status cell", () => {
    expect(source).not.toContain("handleStatusClick: handleViewRuntimeDetail")
    expect(source).not.toContain("statusActionLabel: translations.actions.runtimeDetail")
    expect(source).toContain("handleViewRuntimeDetail")
  })

  it("keeps high-churn table selection as controlled ids instead of pushing selected row objects upstream", () => {
    expect(source).toContain("const [rowSelection, setRowSelection] = React.useState<Record<string, boolean>>({})")
    expect(source).toContain("selectedScans = React.useMemo")
    expect(source).toContain("onSelectionClear")
    expect(source).not.toContain("setSelectedScans")
  })

  it("keeps scan history column definitions independent from the actions object identity", () => {
    expect(source).toContain("handleDelete: handleDeleteScan")
    expect(source).toContain("handleStop: handleStopScan")
    expect(source).not.toContain("handleStatusClick: handleViewRuntimeDetail")
    expect(source).not.toContain("[formatDate, translations, hideTargetColumn, actions]")
  })

  it("hard-cuts scan history query controls to BusinessListQuery shared helpers", () => {
    expect(source).toContain("export type ScanStatusFilter = ScanStatus[]")
    expect(source).toContain("createBusinessListQuery")
    expect(source).toContain("applyBusinessListControlChange")
    expect(source).toContain("setBusinessListPage")
    expect(source).toContain("compileBusinessListFilter")
    expect(source).toContain("compileBusinessListOrderBy")
    expect(source).toContain("toggleBusinessListSorting")
    expect(source).toContain("SCAN_HISTORY_FILTER_FIELDS")
    expect(source).toContain('search: { field: "targetName", operator: "=" }')
    expect(source).toContain('status: { field: "status", operator: "==" }')
    expect(source).toContain("SCAN_HISTORY_SORTABLE_FIELDS")
    expect(source).toContain('createdAt: { orderBy: "createdAt", firstDirection: "desc" }')
    expect(source).toContain("SCAN_HISTORY_DEFAULT_SORTING")
    expect(source).toContain("const [query, setQuery] = React.useState(() => createBusinessListQuery")
    expect(source).toContain("const [pageTokens, setPageTokens]")
    expect(source).toContain("const statusFilter = query.filters.status ?? []")
    expect(source).toContain("const filterParam = compileBusinessListFilter")
    expect(source).toContain("const orderByParam = compileBusinessListOrderBy")
    expect(source).toContain("filter: filterParam")
    expect(source).toContain("orderBy: orderByParam")
    expect(source).toContain("useCursorPaginationScopeChange")
    expect(source).toContain("const pageToken = hasCursorScopeChanged ? undefined : query.pageToken")
    expect(source).toContain("pageToken,")
    expect(source).toContain("isPlaceholderData || hasCursorScopeChanged")
    expect(source).toContain("target: targetId")
    expect(source).toContain("handleSortingChange")
    expect(source).toContain('sortingMode: "server" as const')
    expect(source).not.toContain("function buildStatusFilterQuery")
    expect(source).not.toContain("function combineScanHistoryFilterQuery")
    expect(source).not.toContain("statusParam")
    expect(source).not.toContain("search: statusFilter.length")
    expect(source).not.toContain('React.useState<ScanStatus | "all">("all")')
  })
})
