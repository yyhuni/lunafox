import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/target/all-targets-detail-view-sections.tsx"), "utf8")
const legacyBulkScanExport = ["Bulk", "Initiate", "Scan", "Sheet"].join("")

describe("all-targets-detail-view-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function AllTargetsDetailViewLoadingState")
    expect(source).toContain("className")
    expect(source).toContain("dynamic(")
    expect(source).toContain("useDeferredInteractionMount")
    expect(source).toContain("loading: () => null")
  })

  it("routes initial target loading through the real targets table owner", () => {
    expect(source).toContain("state: AllTargetsDetailViewState")
    expect(source).toContain("<TargetsDataTable")
    expect(source).toContain("loading")
    expect(source).toContain("loadingRowCount={rowCount}")
    expect(source.match(/stableSurfaceRowCount={rowCount}/g)).toHaveLength(1)
    expect(source.match(/stableSurfaceRowCount={stableSurfaceRowCount}/g)).toHaveLength(1)
    expect(source).not.toContain("TargetsDataTableSkeleton")
  })

  it("keeps target loading and resolved tables in natural flow", () => {
    expect(source).not.toContain("fillAvailableHeight")
  })

  it("keeps the same named shared-table regions for loading and resolved target states", () => {
    expect(source).toContain("const ALL_TARGETS_LOADING_SLOTS")
    expect(source).toContain('toolbar: "all-targets-toolbar"')
    expect(source).toContain('body: "all-targets-table-body"')
    expect(source).toContain('pagination: "all-targets-pagination"')
    expect(source.match(/loadingSlots={ALL_TARGETS_LOADING_SLOTS}/g)).toHaveLength(2)
  })

  it("preloads the add-target sidebar from hover intent and releases hidden drawers after close", () => {
    expect(source).toContain("deferredInteractionUnmountDelayMs")
    expect(source).toContain("const sidebarMountOptions = { unmountDelayMs: deferredInteractionUnmountDelayMs }")
    expect(source).toContain("preloadWhen: state.shouldPrefetchOrgs")
    expect(source).toContain("useDeferredInteractionMount(state.initiateScanDialogOpen, sidebarMountOptions)")
    expect(source).toContain("useDeferredInteractionMount(state.bulkInitiateScanDialogOpen, sidebarMountOptions)")
    expect(source).toContain("useDeferredInteractionMount(state.scheduleScanDialogOpen, sidebarMountOptions)")
  })

  it("keeps bulk target delete confirmation summary-only by default", () => {
    expect(source).toContain("bulkDeleteDialogOpen")
    expect(source).toContain("bulkDeleteTargetMessage")
    expect(source).toContain("confirmDelete")
    expect(source).not.toContain("state.selectedTargets.map((target) => (")
  })

  it("wires selected targets into the bulk initiate scan drawer", () => {
    expect(source).toContain("BulkInitiateScanDrawer")
    expect(source).not.toContain(legacyBulkScanExport)
    expect(source).toContain("onBulkInitiateScan={state.handleBulkInitiateScan}")
    expect(source).toContain("selectedRows={state.selectedTargets}")
    expect(source).toContain("targetIds={state.selectedTargets.map((target) => target.id)}")
    expect(source).toContain('state.tScanInitiate("bulkTargetsDesc"')
  })
})
