import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scheduled/scheduled-scan-data-table.tsx"), "utf8")

describe("scheduled-scan-data-table contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ScheduledScanDataTable")
    expect(source).toContain("from \"@tanstack/react-table\"")
  })

  it("uses the custom scheduled scan toolbar instead of the generic table toolbar", () => {
    expect(source).toContain("ScheduledScanTableToolbar")
    expect(source).toContain("hideToolbar: true")
    expect(source).toContain("searchAfter")
    expect(source).toContain("enableRowSelection")
  })

  it("routes selected scheduled scan delete through the shared selected-row confirmation", () => {
    expect(source).toContain("onBulkDelete?: () => void")
    expect(source).toContain("onSelectionChange?: (selectedRows: ScheduledScan[]) => void")
    expect(source).toContain("selectedRows?: ScheduledScan[]")
    expect(source).toContain("onBulkDelete,")
    expect(source).toContain("deleteConfirmation: onBulkDelete")
    expect(source).toContain('state.tConfirm("bulkDeleteScheduledScanTitle")')
    expect(source).toContain('state.tConfirm("bulkDeleteScheduledScanMessage"')
    expect(source).toContain('state.tConfirm("confirmDelete")')
    expect(source).not.toContain("showBulkDelete: false")
  })

  it("accepts caller-owned selected-row status actions and forwards them to UnifiedDataTable", () => {
    expect(source).toContain("selectedRowActions?: SelectedRowActionBarAction[]")
    expect(source).toContain("selectedRowActions,")
    expect(source).toContain("actions={{")
  })

  it("renders quick filters with the shared Tabs component instead of custom tab buttons", () => {
    expect(source).toContain("ScheduledScanFilterTabs")
    expect(source).toContain("TabsList")
    expect(source).toContain("TabsTrigger")
    expect(source).toContain("onValueChange")
    expect(source).not.toContain("aria-pressed={active}")
    expect(source).toContain("const quickFilters")
    expect(source).not.toContain("\"today\"")
    expect(source).not.toContain("\"highFrequency\"")
    expect(source).not.toContain("\"organization\"")
    expect(source).not.toContain("\"target\"")
  })

  it("shows quick filter count badges in the scheduled scan tabs", () => {
    expect(source).toContain("quickFilterCounts")
    expect(source).toContain("TabsCountBadge")
    expect(source).toContain("aria-label={typeof count === \"number\"")
    expect(source).not.toContain('variant="filterCount"')
    expect(source).not.toContain("min-w-[1.25rem]")
  })

  it("keeps quick filters and create actions on one compact desktop toolbar row", () => {
    expect(source).toContain("lg:flex-row lg:items-start lg:justify-between")
    expect(source).not.toContain("lg:flex-row lg:items-center lg:justify-between")
    expect(source).not.toContain("xl:flex-row xl:items-center xl:justify-between")
  })

  it("keeps scheduled-scan creation as an ordinary dense toolbar Button", () => {
    expect(source).toContain('<Button type="button" size="sm" onClick={onAddNew}>')
    expect(source).not.toContain("PageActionButton")
  })

  it("does not keep a duplicate scheduled scan filter dropdown alongside the filter tabs", () => {
    expect(source).not.toContain("DropdownMenuTrigger")
    expect(source).not.toContain("filterButtonLabel")
    expect(source).not.toContain("Filter, Plus")
  })

  it("passes loading ownership into the shared table body", () => {
    expect(source).toContain("loading = false")
    expect(source).toContain("initialLoading?: boolean")
    expect(source).toContain("initialLoading = false")
    expect(source).toContain("loadingRowCount")
    expect(source).toContain("loadingRowHeightEstimate")
    expect(source).toContain("loading,")
    expect(source).toContain("loadingRowCount,")
    expect(source).toContain("loadingRowHeightEstimate,")
    expect(source).toContain('loadingPresentation: initialLoading ? "initial" : "rows"')
  })

  it("replaces the external toolbar with non-interactive initial chrome", () => {
    expect(source).toContain('data-slot="scheduled-scan-table-toolbar"')
    expect(source).toContain('data-loading-presentation={initialLoading ? "initial" : undefined}')
    expect(source).toContain("inert={initialLoading ? true : undefined}")
    expect(source).toContain('<SearchToolbarSkeleton toolbarDensity="compact" />')
    expect(source).toContain('<ActionSkeleton size="sm" widthClassName="w-24" />')
    expect(source).toContain("disabled={initialLoading}")
  })
})
