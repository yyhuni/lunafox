import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/workflow/scan-workflow-page.tsx"), "utf8")

describe("scan-workflow-page contract", () => {
  it("keeps the workflow list loading owned by the resolved management table", () => {
    expect(source).toContain("export function ScanWorkflowPageLoadingState")
    expect(source).toContain("ContentHandoff")
    expect(source).toContain("BusinessListDataTable")
    expect(source).toContain("createWorkflowManagementColumns")
    expect(source).toContain('data-slot="scan-workflow-page-loading-state"')
    expect(source).toContain("loadingRowHeightEstimate: 50.5")
    expect(source).toContain("function ScanWorkflowToolbar")
    expect(source).toContain("function ScanWorkflowPrimaryRegion")
    expect(source).toContain('getLoadingStructureSlotAttributes("scan-workflow-toolbar")')
    expect(source).toContain('getLoadingStructureSlotAttributes("scan-workflow-primary-region")')
    expect(source).not.toContain("loading={loading}")
  })

  it("opens the composition builder from the row action without a detail drawer", () => {
    expect(source).toContain("const [editingWorkflow, setEditingWorkflow]")
    expect(source).toContain("const openBuilder")
    expect(source).toContain("onEdit: openBuilder")
    expect(source).toContain("WorkflowCompositionCanvas")
    expect(source).toContain('onBack={() => setView("management")}')
    expect(source).not.toContain("DetailDrawer")
    expect(source).not.toContain("WorkflowDetailDrawer")
    expect(source).not.toContain("YamlViewer")
    expect(source).not.toContain("onRowClick:")
  })

  it("keeps scan configuration navigation on management views and out of the focused builder", () => {
    expect(source).toContain('import { ScanConfigurationWorkspace } from "@/components/scan/scan-configuration-workspace"')
    expect(source).toContain('<ScanConfigurationWorkspace activeTab="workflows">')
    const builderStart = source.indexOf('{view === "builder" ? (')
    const managementStart = source.indexOf(') : (', builderStart)
    const builderSource = source.slice(builderStart, managementStart)

    expect(builderSource).toContain("<WorkflowCompositionCanvas")
    expect(builderSource).not.toContain("ScanConfigurationWorkspace")
  })

  it("drives the builder engine library from the hook-owned catalog", () => {
    expect(source).toContain('useEngineCatalog(view === "builder")')
    expect(source).toContain("buildWorkflowEngineLibrary(engineCatalog.data, locale)")
    expect(source).toContain("engines={workflowEngineLibrary}")
    expect(source).toContain("engineCatalogStatus={workflowEngineCatalogStatus}")
    expect(source).not.toContain("ENGINE_LIBRARY")
  })

  it("does not expose delete or bulk-selection workflow actions", () => {
    expect(source).not.toContain("ConfirmDialog")
    expect(source).not.toContain("onDelete:")
    expect(source).toContain("showBulkDelete: false")
    expect(source).toContain("readOnly={editingWorkflow?.item.isBuiltin === true}")
  })

  it("keeps the management table compact while preserving the creation action", () => {
    expect(source).toContain("SimpleSearchToolbar")
    expect(source).toContain("<ScanWorkflowToolbar")
    expect(source).toContain("<ScanWorkflowPrimaryRegion>")
    expect(source).toContain("const toolbarRight")
    expect(source).toContain('import { Button } from "@/components/ui/button"')
    expect(source).toContain('tWorkflow("createWorkflow")')
    expect(source).toContain("showColumnVisibility: false")
    expect(source).not.toContain("DataTableFacetedFilter")
  })

  it("uses only active cursor response availability and resets the query shape before effects commit", () => {
    expect(source).toContain("useCursorPaginationScopeChange")
    expect(source).toContain("const cursorScopeKey = `${workflowFilter ?? \"\"}\\u0000${pagination.pageSize}`")
    expect(source).toContain("const activePageIndex = hasCursorScopeChanged ? 0 : pagination.pageIndex")
    expect(source).toContain("const activePageToken = hasCursorScopeChanged ? undefined : pageTokens[pagination.pageIndex]")
    expect(source).toContain("getCurrentCursorNextPageToken")
    expect(source).toContain("isPlaceholderData || hasCursorScopeChanged")
    expect(source).toContain("paginationNavigation")
    expect(source).toContain("cursorPaginationSummary")
    expect(source).not.toContain("Math.ceil((workflowPage?.totalSize")
  })
})
