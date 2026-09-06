import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/organization-list-sections.tsx"), "utf8")
const legacyBulkScanExport = ["Bulk", "Initiate", "Scan", "Sheet"].join("")

describe("organization-list-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function OrganizationListLoadingState")
    expect(source).toContain("ConfirmDialog")
    expect(source).toContain("dynamic(")
    expect(source).toContain("useDeferredInteractionMount")
    expect(source).toContain("loading: () => null")
  })

  it("routes initial list loading through the real organization table owner", () => {
    expect(source).toContain("state: OrganizationListState")
    expect(source).toContain("<OrganizationDataTable")
    expect(source).toContain("loading")
    expect(source).toContain("loadingRowCount={rowCount}")
    expect(source).not.toContain("fillAvailableHeight")
    expect(source).not.toContain("OrganizationDataTableSkeleton")
  })

  it("releases non-route-critical sidebars after close through the shared lifecycle owner", () => {
    expect(source).toContain("deferredInteractionUnmountDelayMs")
    expect(source).toContain("const sidebarMountOptions = { unmountDelayMs: deferredInteractionUnmountDelayMs }")
    expect(source).toContain("useDeferredInteractionMount(state.addDialogOpen, sidebarMountOptions)")
    expect(source).toContain("useDeferredInteractionMount(state.initiateScanDialogOpen, sidebarMountOptions)")
    expect(source).toContain("useDeferredInteractionMount(state.scheduleScanDialogOpen, sidebarMountOptions)")
  })

  it("renders organization details in the shared right-side drawer from row selection", () => {
    expect(source).toContain("OrganizationDetailDrawer")
    expect(source).toContain("onViewDetail={state.handleViewDetail}")
    expect(source).toContain("OrganizationDetailDrawerView")
    expect(source).toContain('organizationId={String(organization?.id ?? "")}')
    expect(source).toContain("onOpenChange={state.handleDetailOpenChange}")
    expect(source).toContain("previewDescription={organization?.description}")
  })

  it("preloads only high-intent organization detail and create-panel chunks", () => {
    expect(source).toContain("const loadOrganizationDetailDrawer")
    expect(source).toContain("const loadAddOrganizationDialog")
    expect(source).toContain("function preloadOrganizationDetailDrawer")
    expect(source).toContain("function preloadAddOrganizationDialog")
    expect(source).toContain("onDetailIntent={preloadOrganizationDetailDrawer}")
    expect(source).toContain("onAddIntent={preloadAddOrganizationDialog}")
    expect(source).not.toContain("preloadBulkInitiateScanDrawer")
    expect(source).not.toContain("preloadCreateScheduledScanSheet")
  })

  it("keeps bulk destructive confirmation focused on count and consequence instead of an expanded item list", () => {
    expect(source).toContain("ConfirmDialog")
    expect(source).toContain("bulkDeleteDialogOpen")
    expect(source).toContain("bulkDeleteOrgMessage")
    expect(source).toContain("confirmDelete")
    expect(source).toContain('variant="destructive"')
    expect(source).toContain('processingText={state.tConfirm("deleting")}')
    expect(source).not.toContain("state.selectedOrganizations.map((organization) => (")
  })

  it("wires selected organizations into the bulk initiate scan drawer", () => {
    expect(source).toContain("BulkInitiateScanDrawer")
    expect(source).not.toContain(legacyBulkScanExport)
    expect(source).toContain("onBulkInitiateScan={state.handleBulkInitiateScan}")
    expect(source).toContain("selectedRows={state.selectedOrganizations}")
    expect(source).toContain("organizationIds={state.selectedOrganizations.map((organization) => organization.id)}")
    expect(source).toContain('state.tScanInitiate("bulkOrganizationsDesc"')
  })
})
