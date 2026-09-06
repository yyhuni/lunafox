import { describe, expect, it } from "vitest"
import { readdirSync, readFileSync, statSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/data-table/unified-data-table.tsx"), "utf8")
const dataTableTypesSource = readFileSync(path.resolve(process.cwd(), "types/data-table.types.ts"), "utf8")
const stickyColumnShellSource = readFileSync(
  path.resolve(process.cwd(), "components/shared/data-table/sticky-column-shell.ts"),
  "utf8"
)
const productionSourceRoots = ["app", "components"].map((root) => path.resolve(process.cwd(), root))
const productionSourceExtensions = new Set([".ts", ".tsx"])
const approvedStickyRightFiles = new Set([
  path.join("components", "organization", "targets", "targets-columns.tsx"),
])
const stickyActionSurfacePattern =
  /sticky\s+right-0[^\n"`']*(?:bg-card|bg-background|bg-secondary|bg-muted|border-l|group-hover:bg|group-data-\[state=selected\]:bg)/

function listProductionSourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const absolutePath = path.join(dir, entry)
    const stat = statSync(absolutePath)

    if (stat.isDirectory()) {
      if (entry === "__tests__" || entry === "node_modules") {
        return []
      }

      return listProductionSourceFiles(absolutePath)
    }

    if (!productionSourceExtensions.has(path.extname(entry))) {
      return []
    }

    return [absolutePath]
  })
}

describe("unified-data-table contract", () => {
  it("keeps the table shell theme-driven while preserving the shared framing", () => {
    expect(source).toContain("rounded-md")
    expect(source).not.toContain("rounded-none")
    expect(source).not.toContain("border-t-2")
    expect(source).toContain("bg-card")
  })

  it("marks the shared shell with an explicit table-shell variant", () => {
    expect(source).toContain('data-table-variant="shell"')
    expect(source).toContain('data-column-layout={columnLayout}')
  })

  it("keeps shared pagination outside the bordered table surface", () => {
    expect(source).toContain('data-slot="data-table-pagination-surface"')
    expect(source).toContain('"rounded-md border border-border bg-card overflow-x-auto"')
    expect(source).toContain("getLoadingStructureSlotAttributes(loadingSlots.body)")
    expect(source).not.toContain('data-slot="data-table-scroll"')
    expect(source).not.toContain('className="shrink-0 border-t border-border py-2"')
  })

  it("keeps normal paginated tables in natural document flow", () => {
    expect(dataTableTypesSource).not.toContain("fillAvailableHeight?: boolean")
    expect(source).not.toContain("fillAvailableHeight")
    expect(source).not.toContain("data-table-height-mode")
    expect(source).not.toContain("md:sticky")
    expect(source).not.toContain("md:overflow-y-auto")
    expect(source).not.toContain("md:max-h-none")
  })

  it("supports a shared fixed-layout path for semantic-width business tables", () => {
    expect(source).toContain('const columnLayout = behavior?.columnLayout ?? "auto"')
    expect(source).toContain('columnLayout === "fixed" && "table-fixed"')
    expect(source).not.toContain('className="caption-bottom text-sm w-full table-fixed"')
  })

  it("keeps column widths non-interactive while preserving the shared header surface", () => {
    expect(source).not.toContain("ColumnResizer")
    expect(source).not.toContain("cursor-col-resize")
  })

  it("keeps semantic flex minimums as the horizontal-overflow boundary", () => {
    expect(source).toContain("allocateSemanticColumnWidths")
    expect(source).toContain("const tableWidthAllocation = React.useMemo")
    expect(source).toContain("const tableMinWidth = tableWidthAllocation.minimumWidth")
    expect(source).toContain("const visibleLeafColumns = React.useMemo")
    expect(source).toContain("minWidth: tableMinWidth")
    expect(source).not.toContain("minWidth: table.getTotalSize()")
  })

  it("uses a fixed-layout colgroup with bounded semantic allocation", () => {
    expect(source).toContain("ResizeObserver")
    expect(source).toContain("const tableColumnWidthStyles = React.useMemo")
    expect(source).toContain("widthPolicy: column.columnDef.meta?.widthPolicy ?? legacyWidthPolicy")
    expect(source).toContain("legacyFillColumnId")
    expect(source).toContain("<colgroup>")
    expect(source).toContain("<col key={visibleLeafColumns[index]?.id ?? index} style={{ width }} />")
  })

  it("owns the production business-list row rhythm through shared table constants", () => {
    expect(dataTableTypesSource).toContain("rowDensity?: DataTableRowDensity")
    expect(source).toContain('const rowDensity = ui?.rowDensity ?? "dense"')
    expect(source).toContain("data-row-rhythm={rowDensity}")
    expect(source).toContain("TABLE_DENSE_ROW_CLASS")
    expect(source).toContain("TABLE_DENSE_CELL_RHYTHM_CLASS")
    expect(source).toContain("TABLE_DENSE_ROW_RHYTHM_HEIGHT_PX")
    expect(source).toContain("TABLE_DENSE_ROW_ESTIMATED_HEIGHT_PX")
    expect(source).toContain("TABLE_COMFORTABLE_ROW_CLASS")
    expect(source).toContain("TABLE_COMFORTABLE_CELL_RHYTHM_CLASS")
    expect(source).toContain("TABLE_COMFORTABLE_ROW_RHYTHM_HEIGHT_PX")
    expect(source).toContain("TABLE_COMFORTABLE_ROW_ESTIMATED_HEIGHT_PX")
    expect(source).toContain("rhythmHeight: TABLE_DENSE_ROW_RHYTHM_HEIGHT_PX")
    expect(source).toContain("rhythmHeight: TABLE_COMFORTABLE_ROW_RHYTHM_HEIGHT_PX")
    expect(source).toContain("estimateSize: () => rowRhythm.estimatedHeight")
    expect(source).not.toContain("estimateSize: () => 53")
  })

  it("owns virtual scroll height through a shared data-table constant", () => {
    expect(source).toContain("DATA_TABLE_VIRTUAL_SCROLL_CLASSNAME")
    expect(source).not.toContain("max-h-[600px]")
  })

  it("supports sticky critical columns through shared column metadata", () => {
    expect(source).toContain("getStickyColumnClassName")
    expect(source).toContain('from "./sticky-column-shell"')
    expect(source).toContain("getDataTableStickyColumnShellClassName")
    expect(source).not.toContain("sticky right-0 z-20 border-l border-border")
    expect(source).not.toContain("sticky right-0 z-10 border-l border-border")
    expect(source).toContain("header.column.columnDef.meta?.stickyRight")
    expect(source).toContain("cell.column.columnDef.meta?.stickyRight")
  })

  it("keeps sticky action columns visually merged with their table section surface", () => {
    expect(stickyColumnShellSource).toContain('"sticky right-0 z-20 bg-inherit"')
    expect(stickyColumnShellSource).toContain('"sticky right-0 z-10 bg-inherit"')
    expect(stickyColumnShellSource).not.toMatch(/\bbg-(card|background|secondary|muted)\b/)
    expect(stickyColumnShellSource).not.toContain("border-l")
    expect(stickyColumnShellSource).not.toContain("group-hover:bg-")
    expect(stickyColumnShellSource).not.toContain("group-data-[state=selected]:bg-")
  })

  it("rejects page-local sticky action column surface patches", () => {
    const offenders = productionSourceRoots
      .flatMap(listProductionSourceFiles)
      .filter((filePath) => !filePath.endsWith(path.join("components", "shared", "data-table", "sticky-column-shell.ts")))
      .filter((filePath) => stickyActionSurfacePattern.test(readFileSync(filePath, "utf8")))
      .map((filePath) => path.relative(process.cwd(), filePath))

    expect(offenders).toEqual([])
  })

  it("keeps stickyRight usage explicitly reviewed instead of defaulting action columns to sticky", () => {
    const stickyRightUsers = productionSourceRoots
      .flatMap(listProductionSourceFiles)
      .filter((filePath) => readFileSync(filePath, "utf8").includes("stickyRight: true"))
      .map((filePath) => path.relative(process.cwd(), filePath))
      .sort()

    expect(stickyRightUsers).toEqual(Array.from(approvedStickyRightFiles).sort())
  })

  it("passes refresh action configuration into the shared table actions", () => {
    expect(source).toContain("onRefresh")
    expect(source).toContain("refreshLabel")
    expect(source).toContain("isRefreshing")
  })

  it("defaults server-paginated tables without an explicit sorting mode to no sorting affordance", () => {
    expect(source).toContain('const sortingMode = state?.sortingMode ?? (paginationInfo ? "none" : "client")')
    expect(source).toContain("paginationInfo")
  })

  it("keeps business-list chrome behind shared table owners", () => {
    expect(source).toContain("<TableToolbar")
    expect(source).toContain("<TableActions")
    expect(source).toContain("<DataTablePagination")
    expect(source).toContain("showColumnVisibility={showColumnVisibility}")
    expect(source).not.toContain("function renderPagination")
  })

  it("owns loading table rows through the same shared table structure", () => {
    const loadingRowsStart = source.indexOf("const loadingRows = isLoading")
    const loadingRowsBlock = source.slice(
      loadingRowsStart,
      source.indexOf("return (", loadingRowsStart)
    )

    expect(source).toContain("ui?.loading")
    expect(source).toContain("ui?.loadingRowCount")
    expect(source).toContain("ui?.loadingRowHeightEstimate")
    expect(source).toContain("getLoadingRowHeight(rowIndex)")
    expect(source).toContain("return loadingRowHeightEstimate")
    expect(source).toContain("loadingRows")
    expect(loadingRowsBlock).toContain('key={`loading-${rowIndex}`}')
    expect(loadingRowsBlock).toContain("visibleLeafColumns.map")
    expect(loadingRowsBlock).toContain("<TableCell")
    expect(loadingRowsBlock).toContain("<Skeleton")
    expect(loadingRowsBlock).toContain("style={{ height: getLoadingRowHeight(rowIndex) }}")
    expect(loadingRowsBlock).toContain("rowRhythm.rowClassName")
    expect(loadingRowsBlock).toContain("rowRhythm.cellClassName")
    expect(loadingRowsBlock).not.toContain("TABLE_DENSE_ROW_ESTIMATED_HEIGHT_PX")
  })

  it("exposes one shared structure-slot trio alongside a loading-only stable surface", () => {
    expect(dataTableTypesSource).toContain("export interface DataTableLoadingSlots")
    expect(dataTableTypesSource).toContain("loadingSlots?: DataTableLoadingSlots")
    expect(source).toContain("DEFAULT_DATA_TABLE_LOADING_SLOTS")
    expect(source).toContain('toolbar: "data-table-toolbar"')
    expect(source).toContain('body: "data-table-body"')
    expect(source).toContain('pagination: "data-table-pagination"')
    expect(source).toContain("getLoadingStructureSlotAttributes(loadingSlots.toolbar)")
    expect(source).toContain("getLoadingStructureSlotAttributes(loadingSlots.body)")
    expect(source).toContain("getLoadingStructureSlotAttributes(loadingSlots.pagination)")
    expect(source).toContain('throw new Error("loadingSlots requires three unique non-empty slot names.")')
    expect(dataTableTypesSource).toContain("stableSurfaceRowCount?: number")
    expect(source).toContain("TABLE_HEADER_RHYTHM_HEIGHT_PX")
    expect(source).toContain("!isLoading || stableSurfaceRowCount === undefined")
    expect(source).toContain("stableSurfaceRowCount * rowRhythm.rhythmHeight")
    expect(source).toContain('data-stable-surface-row-count={stableSurfaceRowCount}')
    expect(source).toContain('throw new Error("stableSurfaceRowCount requires a positive integer.")')
  })

  it("owns a non-interactive initial table loading presentation", () => {
    expect(dataTableTypesSource).toContain('loadingPresentation?: "rows" | "initial"')
    expect(source).toContain('const isInitialLoading = isLoading && loadingPresentation === "initial"')
    expect(source).toContain("InitialDataTableChromePlaceholder")
    expect(source).toContain("CompactPaginationSkeleton")
    expect(source).toContain('mode={paginationNavigation?.mode === "cursor" ? "cursor" : "numbered"}')
    expect(source).toContain("showSummary={paginationNavigation?.mode !== \"cursor\" || Boolean(cursorPaginationSummary)}")
    expect(source).toContain('buttonCount={paginationNavigation?.mode === "cursor" ? 3 : 4}')
    expect(source).toContain('aria-busy={isLoading || undefined}')
    expect(source).toContain('data-loading-presentation={isLoading ? loadingPresentation : undefined}')
    expect(source).toContain("isInitialLoading ? (")
  })

  it("keeps initial filter slots beside the search slot so narrow toolbars wrap like resolved content", () => {
    const placeholderStart = source.indexOf("function InitialDataTableChromePlaceholder")
    const placeholderEnd = source.indexOf("function shouldIgnoreRowActivation", placeholderStart)
    const placeholderSource = source.slice(placeholderStart, placeholderEnd)

    expect(placeholderSource).toContain("<SearchToolbarSkeleton")
    expect(placeholderSource).toContain("Array.from({ length: filterCount })")
    expect(placeholderSource).not.toContain("after={")
  })

  it("does not render resolved spacer rows for sparse or empty pages", () => {
    expect(source).not.toContain("preserveEmptyStatePageSlots")
    expect(source).not.toContain("data-table-spacer-row")
    expect(source).not.toContain("stableSpacerRows")
    expect(source).not.toContain("spacerRowCount")
  })

  it("routes selected-row bulk delete through the shared selected-row action bar", () => {
    expect(source).toContain("SelectedRowActionBar")
    expect(source).toContain("configuredSelectedRowActions")
    expect(source).toContain("...configuredSelectedRowActions")
    expect(source).toContain("selectedRowActions")
    expect(source).toContain("handleSelectedBulkDeleteClick")
    expect(source).toContain("onClearSelection={handleClearSelection}")
    expect(source).toContain('useTranslations("dataTable")')
    expect(source).toContain('tDataTable("selected", { count: selectedCount })')
    expect(source).toContain('tDataTable("deselectAll")')
    expect(source).not.toContain("showBulkDelete={false}")
    expect(source).not.toContain("showBulkDelete={showBulkDelete}")
    expect(source).not.toContain("onBulkDelete={onBulkDelete}")
    expect(source).not.toContain("deleteConfirmation={deleteConfirmation}")
  })

  it("syncs external selected-row clears back into the internal table selection", () => {
    expect(source).toContain("const externalSelectedRows = state?.selectedRows")
    expect(source).toContain("const externalSelectedRowsCount = externalSelectedRows?.length")
    expect(source).toContain("const prevExternalSelectedRowsCountRef = React.useRef(externalSelectedRowsCount)")
    expect(source).toContain("const prevExternalSelectedRowsCount = prevExternalSelectedRowsCountRef.current")
    expect(source).toContain("prevExternalSelectedRowsCount > 0")
    expect(source).toContain("externalSelectedRowsCount === 0")
    expect(source).toContain("selectedCount > 0")
    expect(source).toContain("table.resetRowSelection()")
  })

  it("keeps row-detail activation accessible and ignores nested controls", () => {
    expect(source).toContain("shouldIgnoreRowActivation")
    expect(source).toContain("handleRowKeyDown")
    expect(source).toContain('tabIndex={onRowClick ? 0 : undefined}')
    expect(source).toContain('aria-label={getRowActionLabel?.(row.original)}')
    expect(source).toContain('event.key === "Enter" || event.key === " "')
    expect(source).toContain('"button"')
    expect(source).toContain('"a"')
    expect(source).toContain('"input"')
    expect(source).toContain('"select"')
    expect(source).toContain('"textarea"')
    expect(source).toContain("[role='checkbox']")
    expect(source).toContain("[data-row-click-exempt='true']")
    expect(source).toContain("event.preventDefault()")
  })

  it("prevents portal dropdown row actions from re-triggering row detail activation", () => {
    expect(source).toContain("[role='menuitem']")
    expect(source).toContain("[role='menuitemcheckbox']")
    expect(source).toContain("[role='menuitemradio']")
    expect(source).toContain("[data-slot='dropdown-menu-content']")
    expect(source).toContain("[data-slot='dropdown-menu-item']")
    expect(source).toContain("[data-slot='dropdown-menu-checkbox-item']")
    expect(source).toContain("[data-slot='dropdown-menu-radio-item']")
  })

  it("keeps generated-file actions named as exports", () => {
    expect(source).toContain("actions?.exportOptions")
    expect(source).toContain("exportOptions={exportOptions}")
    expect(source).not.toContain("downloadOptions")
  })
})
