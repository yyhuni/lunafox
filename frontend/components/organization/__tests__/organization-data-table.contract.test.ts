import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/organization-data-table.tsx"), "utf8")

describe("organization-data-table contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function OrganizationDataTable")
    expect(source).toContain("BusinessListDataTable")
    expect(source).not.toContain("from \"@/components/shared/data-table/unified-data-table\"")
  })

  it("renders selected-row scan and delete actions through the shared action bar", () => {
    expect(source).toContain("SelectedRowActionBar")
    expect(source).toContain("onBulkInitiateScan")
    expect(source).toContain('label: state.tTooltips("initiateScan")')
    expect(source).toContain('group: "scan"')
    expect(source).toContain('group: "danger"')
    expect(source).toContain("onClick: onBulkDelete")
    expect(source).toContain("onClearSelection")
    expect(source).not.toContain('role="toolbar"')
    expect(source).not.toContain('bulkDeleteLabel: state.tActions("delete")')
    expect(source).not.toContain('bulkDeleteLabel: state.tActions("bulkDelete")')
  })

  it("wires row click and intent to the organization detail drawer handler", () => {
    expect(source).toContain("onViewDetail,")
    expect(source).toContain("onDetailIntent,")
    expect(source).toContain("onRowClick: onViewDetail")
    expect(source).toContain("onRowIntent: onDetailIntent")
  })

  it("preloads the create panel from the shared add-control intent signal", () => {
    expect(source).toContain("onAddIntent,")
    expect(source).toContain("onAddHover: onAddIntent")
  })

  it("hides the column visibility control for the compact organization list", () => {
    expect(source).toContain("showColumnVisibility: false")
  })

  it("uses the shared comfortable row rhythm for organization identity cells", () => {
    expect(source).toContain('rowDensity: "comfortable"')
  })

  it("lets the merged organization identity column absorb spare width", () => {
    expect(source).toContain('expandColumnIds: ["name"]')
    expect(source).not.toContain('expandColumnIds: ["name", "description"]')
  })

  it("uses server sorting instead of default current-page sorting", () => {
    expect(source).toContain('sortingMode: "server"')
    expect(source).toContain("sorting,")
    expect(source).toContain("onSortingChange,")
    expect(source).not.toContain("defaultSorting")
  })

  it("passes loading ownership through the shared business-list table", () => {
    expect(source).toContain("loading = false")
    expect(source).toContain("loadingRowCount")
    expect(source).toContain("ORGANIZATION_LIST_LOADING_ROW_HEIGHT_PX = 79")
    expect(source).toContain("loadingRowHeightEstimate: ORGANIZATION_LIST_LOADING_ROW_HEIGHT_PX")
    expect(source).toContain("stableSurfaceRowCount")
    expect(source).toContain("loading,")
    expect(source).toContain("loadingRowCount,")
    expect(source).toContain("stableSurfaceRowCount,")
    expect(source).not.toContain("fillAvailableHeight")
    expect(source).toContain('loadingPresentation: loading ? "initial" : undefined')
  })

  it("keeps the route geometry trio on the shared table in both handoff phases", () => {
    expect(source).toContain("const ORGANIZATION_LIST_LOADING_SLOTS")
    expect(source).toContain('toolbar: "organization-list-toolbar"')
    expect(source).toContain('body: "organization-list-body"')
    expect(source).toContain('pagination: "organization-list-pagination"')
    expect(source).toContain("loadingSlots: ORGANIZATION_LIST_LOADING_SLOTS")
  })

  it("does not opt resolved sparse pages into spacer rows", () => {
    expect(source).not.toContain("preserveEmptyStatePageSlots")
  })
})
