import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-history-list-sections.tsx"), "utf8")

describe("scan-history-list-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ScanHistoryListLoadingState")
    expect(source).toContain("export function ScanHistoryListTable")
    expect(source).toContain("export function ScanHistoryListDialogsSection")
    expect(source).toContain("from \"react\"")
  })

  it("keeps stable localized table chrome visible during component loading", () => {
    expect(source).toContain('target: tColumns("scanHistory.target")')
    expect(source).toContain('executedEngines: tColumns("scanHistory.executedEngines")')
    expect(source).toContain('createdAt: tColumns("common.createdAt")')
    expect(source).toContain('searchPlaceholder={tScan("history.searchPlaceholder")}')
    expect(source).not.toContain('target: ""')
    expect(source).not.toContain('searchPlaceholder=""')
  })

  it("keeps the list skeleton as the single visible loading owner", () => {
    expect(source).toContain("from \"./scan-history-data-table\"")
    expect(source).toContain("createScanHistoryColumns")
    expect(source).toContain("getLoadingOwnerAttributes")
    expect(source).toContain("owner?: string")
    expect(source).toContain("owner !== undefined && !owner.trim()")
    expect(source).toContain("...(owner ? getLoadingOwnerAttributes")
    expect(source).toContain('data-slot="scan-history-list-loading-state"')
    expect(source).toContain("loadingRowCount={rowCount}")
    expect(source).toContain("stableSurfaceRowCount={rowCount}")
    expect(source).toContain("stableSurfaceRowCount={stableSurfaceRowCount}")
    expect(source).toContain("initialLoading")
    expect(source).toContain("hideTargetColumn")
    expect(source).not.toContain("from \"next/dynamic\"")
    expect(source).not.toContain("scan-history-drawer")
    expect(source).not.toContain("scan-history-list-skeleton")
    expect(source).not.toContain("ScanHistoryListSkeleton")
    expect(source).not.toContain("loading: () => <ScanHistoryListSkeleton")
  })

  it("keeps the same named shared-table regions for loading and resolved list states", () => {
    expect(source).toContain("const SCAN_HISTORY_LIST_LOADING_SLOTS")
    expect(source).toContain('toolbar: "scan-history-list-toolbar"')
    expect(source).toContain('body: "scan-history-list-body"')
    expect(source).toContain('pagination: "scan-history-list-pagination"')
    expect(source.match(/loadingSlots={SCAN_HISTORY_LIST_LOADING_SLOTS}/g)).toHaveLength(2)
  })

  it("keeps page-level refresh out of the table toolbar", () => {
    expect(source).not.toContain("onRefresh=")
    expect(source).not.toContain("refreshLabel=")
    expect(source).not.toContain("isRefreshing=")
    expect(source).toContain('bulkDeleteLabel={state.tCommon("actions.delete")}')
    expect(source).toContain("selectedRowActions={state.hideToolbar ? undefined : state.selectedRowActions}")
    expect(source).not.toContain('bulkDeleteLabel={state.tCommon("actions.bulkDelete")}')
  })

  it("passes server sorting state through both resolved and loading tables", () => {
    expect(source).toContain("sortingMode={state.sortingMode}")
    expect(source).toContain("sorting={state.sorting}")
    expect(source).toContain("onSortingChange={state.handleSortingChange}")
    expect(source).toContain('sortingMode="server"')
  })

  it("opens runtime details from the whole table row", () => {
    expect(source).toContain("onRowClick={state.handleViewRuntimeDetail}")
    expect(source).toContain('rowActionLabel={state.tScan("history.runtimeDrawer.open")}')
  })
})
