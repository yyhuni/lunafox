import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-history-list.tsx"), "utf8")

describe("scan-history-list contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ScanHistoryList")
    expect(source).toContain("from \"./scan-history-list-sections\"")
  })

  it("uses the shared skeleton handoff instead of a hard initial loading branch", () => {
    expect(source).toContain("ContentHandoff")
    expect(source).toContain("const isInitialLoading = state.isLoading && !state.scans.length")
    expect(source).toContain("if (!isInitialLoading && state.error)")
    expect(source).toContain('owner="scan-history-list-view-content"')
    expect(source).toContain('layer = "workspace"')
    expect(source).toContain('layer={layer}')
    expect(source).toContain("skeleton={")
    expect(source).toContain("<ScanHistoryListLoadingState")
    expect(source).not.toContain('owner="scan-history-list-view"')
    expect(source).toContain("<ScanHistoryListTable state={state} stableSurfaceRowCount={loadingRowCount} />")
    expect(source).toContain("<ScanHistoryListDialogsSection state={state} />")
    expect(source).toContain('from "@/components/shared/feedback/app-error-state"')
    expect(source).toContain("<AppErrorState")
  })

  it("sizes initial table skeleton rows from the active page size with a first-screen cap", () => {
    expect(source).toContain("getDataTableSkeletonRowCount")
    expect(source).toContain("const loadingRowCount = getDataTableSkeletonRowCount(state.pagination.pageSize)")
    expect(source).toContain("rowCount={loadingRowCount}")
    expect(source).toContain("hideTargetColumn={hideTargetColumn}")
    expect(source).toContain("layer={layer}")
    expect(source).not.toContain("rowCount={customPageSize ?? 8}")
  })
})
