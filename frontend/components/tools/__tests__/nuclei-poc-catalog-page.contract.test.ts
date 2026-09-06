import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/tools/nuclei-poc-catalog-page.tsx"), "utf8")

type NucleiCatalogMessages = {
  pages: {
    nucleiCatalog: {
      actions: Record<string, string>
      batch: Record<string, string>
      selected: Record<string, string>
    }
  }
  toast: {
    nucleiPoc: {
      activation: Record<string, string>
    }
  }
}

describe("nuclei-poc-catalog-page contract", () => {
  it("uses the backend-owned service and shared catalog owners", () => {
    expect(source).toContain("BusinessListDataTable")
    expect(source).toContain("DetailDrawer")
    expect(source).toContain("DataTableFacetPanel")
    expect(source).toContain("NucleiPocSyncDialog")
    expect(source).toContain("NucleiPocSyncSourceStatus")
    expect(source).toContain("useNucleiPocs")
    expect(source).toContain("useNucleiPocFilterOptions")
    expect(source).toContain("useNucleiPocSource")
    expect(source).not.toContain("nucleiPocCatalogPreviewItems")
    expect(source).not.toContain("filterNucleiPocCatalog")
  })

  it("uses complete-catalog tag options instead of the current cursor page", () => {
    expect(source).toContain('const tagOptionsQuery = useNucleiPocFilterOptions("tags")')
    expect(source).toContain("tagOptionsQuery.data?.results")
    expect(source).toContain("optionByValue.has(tag)")
    expect(source).not.toContain("count: 0")
    expect(source).not.toContain("items.forEach((item) => item.tags")
  })

  it("keeps POC content read-only while allowing persisted activation", () => {
    expect(source).toContain("enableRowSelection: true")
    expect(source).toContain('accessorKey: "isEnabled"')
    expect(source).toContain('updateMask: ["isEnabled"]')
    expect(source).toContain("useUpdateNucleiPocEnabled")
    expect(source).toContain("selectedRowActions")
    expect(source).toContain("useSetNucleiPocActivation")
    expect(source).not.toContain("openCreateForm")
    expect(source).not.toContain("deleteItems")
    expect(source).not.toContain('accessorKey: "source"')
  })

  it("keeps full-catalog activation inside one shared menu and confirmation owner", () => {
    expect(source).toContain("NucleiPocBatchActivationControls")
    expect(source).toContain('actions.enableAll')
    expect(source).toContain('actions.disableAll')
    expect(source).toContain("AlertDialog")
    expect(source).toContain("useSetNucleiPocActivation")
    expect(source).toContain("activationTarget")
    expect(source).toContain("onConfirm={confirmActivation}")
  })

  it("keeps full-catalog and selected activation copy complete in both supported locales", () => {
    for (const locale of ["zh", "en"] as const) {
      const messages = JSON.parse(readFileSync(path.resolve(process.cwd(), `messages/${locale}.json`), "utf8")) as NucleiCatalogMessages
      const catalog = messages.pages.nucleiCatalog
      const activation = messages.toast.nucleiPoc.activation
      for (const key of ["batch", "enableAll", "disableAll", "enableSelected", "disableSelected", "selectAll", "selectRow"]) expect(catalog.actions[key]).toEqual(expect.any(String))
      for (const key of ["enableTitle", "disableTitle", "enableDescription", "disableDescription", "confirm", "processing"]) expect(catalog.batch[key]).toEqual(expect.any(String))
      for (const key of ["enableTitle", "disableTitle", "description", "processing", "confirm"]) expect(catalog.selected[key]).toEqual(expect.any(String))
      for (const key of ["loading", "success", "noChange", "conflict", "error", "unknown"]) expect(activation[key]).toEqual(expect.any(String))
      expect(catalog.batch.enableDescription).not.toMatch(/\b(count|数量|条)\b/i)
      expect(catalog.batch.disableDescription).not.toMatch(/\b(count|数量|条)\b/i)
      expect(catalog.selected.description).toContain("{count")
    }
  })

  it("shows explicit source selection, task progress, and cursor pagination", () => {
    expect(source).toContain('toolbarRight: <Button')
    expect(source).toContain("getCursorPaginationNavigation")
    expect(source).toContain("getCursorPageTransition")
    expect(source).toContain("crypto.randomUUID")
    expect(source).toContain("isNucleiPocGiteeSyncSourceUrl")
    expect(source).toContain('value: "gitee"')
    expect(source).toContain('value: "custom"')
    expect(source).not.toContain("preparedToast")
    expect(source).not.toContain("previewBoundary")
  })

  it("adopts a canonical conflict task and keeps start-over explicit", () => {
    expect(source).toContain("getNucleiPocQueryActiveSyncTaskName(error)")
    expect(source).toContain("setActiveTaskName(existingTaskName)")
    expect(source).toContain("syncMutation.reset()")
    expect(source).toContain("const startNewSync = () =>")
    expect(source).toContain("setActiveTaskName(null)")
    expect(source).toContain("setSyncSubmitted(false)")
    expect(source).not.toContain("startNewSync = submitSync")
  })

  it("derives every initial loading row count from the active page size", () => {
    expect(source).toContain("const PAGE_SIZE = 10")
    expect(source).toContain("getDataTableSkeletonRowCount(PAGE_SIZE)")
    expect(source).toContain("loadingRowCount: NUCLEI_POC_CATALOG_LOADING_ROW_COUNT")
    expect(source).toContain("stableSurfaceRowCount: NUCLEI_POC_CATALOG_LOADING_ROW_COUNT")
    expect(source).toContain("columns={columns}")
    expect(source).toContain('loadingPresentation: "initial"')
    expect(source).toContain("initialLoadingToolbarFilterCount: 1")
    expect(source).toContain('mode: "cursor"')
    expect(source).toContain("singleBadgeValue: (row) => tSeverity(row.severity)")
    expect(source).toContain("singleBadgeValues: SEVERITY_LEVELS.map((severity) => tSeverity(severity))")
  })

  it("keeps the initial source skeleton aligned with a wrapped committed source", () => {
    expect(source).toContain("function NucleiPocSourceStatusLoadingText")
    expect(source).toContain('data-slot="nuclei-poc-source-status-loading-text"')
    expect(source).toContain("DEFAULT_NUCLEI_TEMPLATES_GIT_URL")
    expect(source).toContain('className={cn("break-all font-mono", textRole.caption)}')
    expect(source).toContain('className="absolute inset-0 radius-badge"')
  })
})
