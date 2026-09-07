import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-history-data-table.tsx"), "utf8")

describe("scan-history-data-table contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ScanHistoryDataTable")
    expect(source).toContain("from \"react\"")
  })

  it("uses the shared route-facing business-list wrapper", () => {
    expect(source).toContain("BusinessListDataTable")
    expect(source).toContain("selectedRowActions")
    expect(source).toContain("selectedRowActions,")
    expect(source).not.toContain("UnifiedDataTable")
  })

  it("auto-sizes stable value columns and keeps width ownership beside each column", () => {
    expect(source).toContain("enableAutoColumnSizing: true")
    expect(source).not.toContain("expandColumnIds")
    expect(source).not.toContain("agentName")
    expect(source).not.toContain("fitTableToContent")
  })

  it("does not opt resolved sparse pages into spacer rows", () => {
    expect(source).not.toContain("preserveEmptyStatePageSlots")
  })

  it("passes loading ownership through the shared business-list table", () => {
    expect(source).toContain("loading = false")
    expect(source).toContain("initialLoading?: boolean")
    expect(source).toContain("initialLoading = false")
    expect(source).toContain("loadingRowCount")
    expect(source).toContain("stableSurfaceRowCount")
    expect(source).toContain("loading,")
    expect(source).toContain("loadingRowCount,")
    expect(source).toContain("stableSurfaceRowCount,")
    expect(source).toContain('loadingPresentation: initialLoading ? "initial" : "rows"')
  })

  it("declares server sorting ownership instead of current-page sorting", () => {
    expect(source).toContain("sortingMode?: DataTableSortingMode")
    expect(source).toContain("sorting?: SortingState")
    expect(source).toContain("onSortingChange?: (sorting: SortingState) => void")
    expect(source).toContain('sortingMode = "none"')
    expect(source).toContain("sortingMode,")
    expect(source).toContain("sorting,")
    expect(source).toContain("onSortingChange,")
    expect(source).not.toContain('sortingMode: "client"')
  })

  it("uses the shared dense row rhythm for both loading and resolved history rows", () => {
    expect(source).not.toContain("SCAN_HISTORY_LOADING_ROW_HEIGHT_VARIABLES")
    expect(source).not.toContain("loadingRowHeightEstimate")
    expect(source).toContain("stableSurfaceRowCount")
  })

  it("exposes an accessible label for whole-row runtime detail activation", () => {
    expect(source).toContain("rowActionLabel?: string")
    expect(source).toContain("getRowActionLabel: onRowClick && rowActionLabel ? () => rowActionLabel : undefined")
  })
})
