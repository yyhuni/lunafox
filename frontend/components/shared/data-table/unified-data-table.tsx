"use client"

import * as React from "react"
import {
  flexRender,
} from "@tanstack/react-table"
import { useVirtualizer } from "@tanstack/react-virtual"
import { useTranslations } from "next-intl"

import { IconTrash } from "@/components/icons"
import {
  TABLE_COMFORTABLE_CELL_RHYTHM_CLASS,
  TABLE_COMFORTABLE_ROW_CLASS,
  TABLE_COMFORTABLE_ROW_ESTIMATED_HEIGHT_PX,
  TABLE_COMFORTABLE_ROW_RHYTHM_HEIGHT_PX,
  TABLE_DENSE_CELL_RHYTHM_CLASS,
  TABLE_DENSE_ROW_CLASS,
  TABLE_DENSE_ROW_ESTIMATED_HEIGHT_PX,
  TABLE_DENSE_ROW_RHYTHM_HEIGHT_PX,
  TABLE_HEADER_RHYTHM_HEIGHT_PX,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { cn } from "@/lib/utils"

import { DataTablePagination } from "./pagination"
import { allocateSemanticColumnWidths } from "./semantic-column-widths"
import { getDataTableStickyColumnShellClassName } from "./sticky-column-shell"
import { useTableState } from "./use-table-state"
import { TableToolbar } from "./table-toolbar"
import { TableActions } from "./table-actions"
import {
  SelectedRowActionBar,
  type SelectedRowActionBarAction,
} from "./selected-row-action-bar"
import {
  AlertDialog,
  AlertDialogClose, AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Skeleton } from "@/components/ui/skeleton"
import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { CompactPaginationSkeleton } from "@/components/shared/loading/compact-pagination-skeleton"
import { getLoadingStructureSlotAttributes } from "@/components/shared/loading/loading-owner"
import { SearchToolbarSkeleton } from "@/components/shared/loading/search-toolbar-skeleton"
import type {
  DataTableLoadingSlots,
  UnifiedDataTableProps,
} from "@/types/data-table.types"

const DATA_TABLE_VIRTUAL_SCROLL_CLASSNAME = "max-h-150 overflow-y-auto"
const DATA_TABLE_STABLE_PAGE_ROW_MAX = 10
const DATA_TABLE_RESIZE_OBSERVER_EPSILON_PX = 1
const DEFAULT_DATA_TABLE_LOADING_SLOTS = {
  toolbar: "data-table-toolbar",
  body: "data-table-body",
  pagination: "data-table-pagination",
} satisfies DataTableLoadingSlots
const ROW_ACTIVATION_EXEMPT_SELECTOR =
  [
    "button",
    "a",
    "input",
    "select",
    "textarea",
    "[role='button']",
    "[role='checkbox']",
    "[role='link']",
    "[role='menu']",
    "[role='menuitem']",
    "[role='menuitemcheckbox']",
    "[role='menuitemradio']",
    "[data-row-click-exempt='true']",
    "[data-slot='dropdown-menu-content']",
    "[data-slot='dropdown-menu-item']",
    "[data-slot='dropdown-menu-checkbox-item']",
    "[data-slot='dropdown-menu-radio-item']",
  ].join(",")

function InitialDataTableChromePlaceholder({
  filterCount,
  actionCount,
  toolbarDensity,
}: {
  filterCount: number
  actionCount: number
  toolbarDensity: "compact" | "standard"
}) {
  const actionSize = toolbarDensity === "standard" ? "default" : "sm"

  return (
    <div
      data-slot="data-table-initial-toolbar-skeleton"
      aria-hidden="true"
      className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between"
    >
      <div className="flex w-full min-w-0 flex-wrap items-center gap-2 sm:flex-1">
        <SearchToolbarSkeleton
          toolbarDensity={toolbarDensity}
        />
        {Array.from({ length: filterCount }).map((_, index) => (
          <ActionSkeleton key={index} size={actionSize} widthClassName="w-20" />
        ))}
      </div>
      {actionCount > 0 ? (
        <div className="flex w-full flex-wrap items-center gap-2 sm:w-auto sm:justify-end">
          {Array.from({ length: actionCount }).map((_, index) => (
            <ActionSkeleton key={index} size={actionSize} widthClassName="w-24" />
          ))}
        </div>
      ) : null}
    </div>
  )
}

function shouldIgnoreRowActivation(event: React.MouseEvent | React.KeyboardEvent) {
  const target = event.target
  return target instanceof Element && Boolean(target.closest(ROW_ACTIVATION_EXEMPT_SELECTOR))
}

/**
 * Unified data table component
 *
 * Features:
 * - Generic support, type safety
 * - Row selection, sorting, column visibility, column resizing
 * - Client/server-side pagination
 * - Simple search/smart filtering
 * - Bulk operations, export functionality
 * - Confirmation dialogs
 * - Virtual scrolling for large datasets (10,000+ rows)
 */
export function UnifiedDataTable<TData>(props: UnifiedDataTableProps<TData>) {
  const { data, columns, getRowId, state, ui, behavior, actions } = props
  const tDataTable = useTranslations("dataTable")

  const externalPagination = state?.pagination
  const setExternalPagination = state?.setPagination
  const paginationInfo = state?.paginationInfo
  const cursorPaginationSummary = state?.cursorPaginationSummary
  const paginationNavigation = state?.paginationNavigation
  const onPaginationChange = state?.onPaginationChange
  const hidePagination = ui?.hidePagination ?? false
  const pageSizeOptions = ui?.pageSizeOptions

  const hideToolbar = ui?.hideToolbar ?? false
  const showColumnVisibility = ui?.showColumnVisibility ?? true
  const toolbarDensity = ui?.toolbarDensity ?? "compact"
  const rowDensity = ui?.rowDensity ?? "dense"
  const rowRhythm = rowDensity === "comfortable"
    ? {
        rowClassName: TABLE_COMFORTABLE_ROW_CLASS,
        cellClassName: TABLE_COMFORTABLE_CELL_RHYTHM_CLASS,
        rhythmHeight: TABLE_COMFORTABLE_ROW_RHYTHM_HEIGHT_PX,
        estimatedHeight: TABLE_COMFORTABLE_ROW_ESTIMATED_HEIGHT_PX,
      }
    : {
        rowClassName: TABLE_DENSE_ROW_CLASS,
        cellClassName: TABLE_DENSE_CELL_RHYTHM_CLASS,
        rhythmHeight: TABLE_DENSE_ROW_RHYTHM_HEIGHT_PX,
        estimatedHeight: TABLE_DENSE_ROW_ESTIMATED_HEIGHT_PX,
      }
  const toolbarLeft = ui?.toolbarLeft
  const toolbarRight = ui?.toolbarRight

  const searchMode = behavior?.searchMode ?? 'simple'
  const searchPlaceholder = ui?.searchPlaceholder
  const searchValue = state?.searchValue
  const onSearch = behavior?.onSearch
  const isSearching = state?.isSearching
  const filterFields = ui?.filterFields
  const filterExamples = ui?.filterExamples

  const enableRowSelection = behavior?.enableRowSelection ?? true
  const externalRowSelection = state?.rowSelection
  const externalOnRowSelectionChange = state?.onRowSelectionChange
  const onSelectionChange = state?.onSelectionChange
  const externalSelectedRows = state?.selectedRows

  const onBulkDelete = actions?.onBulkDelete
  const bulkDeleteLabel = actions?.bulkDeleteLabel ?? "Delete"
  const showBulkDelete = actions?.showBulkDelete ?? true
  const configuredSelectedRowActions = actions?.selectedRowActions ?? []

  const onAddNew = actions?.onAddNew
  const onAddHover = actions?.onAddHover
  const addButtonLabel = actions?.addButtonLabel ?? "Add"
  const showAddButton = actions?.showAddButton ?? true

  const onBulkAdd = actions?.onBulkAdd
  const bulkAddLabel = actions?.bulkAddLabel ?? "Bulk Add"
  const showBulkAdd = actions?.showBulkAdd ?? true

  const onRefresh = actions?.onRefresh
  const refreshLabel = actions?.refreshLabel ?? "Refresh"
  const showRefresh = actions?.showRefresh ?? true
  const isRefreshing = actions?.isRefreshing ?? false

  const exportOptions = actions?.exportOptions

  const externalColumnVisibility = state?.columnVisibility
  const externalOnColumnVisibilityChange = state?.onColumnVisibilityChange

  const externalSorting = state?.sorting
  const externalOnSortingChange = state?.onSortingChange
  const sortingMode = state?.sortingMode ?? (paginationInfo ? "none" : "client")
  const defaultSorting = state?.defaultSorting ?? []

  const emptyMessage = ui?.emptyMessage ?? "No results"
  const emptyComponent = ui?.emptyComponent
  const isLoading = ui?.loading ?? false
  const loadingPresentation = ui?.loadingPresentation ?? "rows"
  const initialLoadingToolbarFilterCount = ui?.initialLoadingToolbarFilterCount ?? 0
  const isInitialLoading = isLoading && loadingPresentation === "initial"
  const loadingRowCount = ui?.loadingRowCount ?? DATA_TABLE_STABLE_PAGE_ROW_MAX
  const loadingRowHeightEstimate = ui?.loadingRowHeightEstimate
  const stableSurfaceRowCount = ui?.stableSurfaceRowCount
  const loadingSlots = ui?.loadingSlots ?? DEFAULT_DATA_TABLE_LOADING_SLOTS
  const loadingSlotValues = [
    loadingSlots.toolbar,
    loadingSlots.body,
    loadingSlots.pagination,
  ]

  if (
    loadingSlotValues.some((slot) => typeof slot !== "string" || !slot.trim()) ||
    new Set(loadingSlotValues).size !== loadingSlotValues.length
  ) {
    throw new Error("loadingSlots requires three unique non-empty slot names.")
  }

  if (
    stableSurfaceRowCount !== undefined &&
    (!Number.isInteger(stableSurfaceRowCount) || stableSurfaceRowCount <= 0)
  ) {
    throw new Error("stableSurfaceRowCount requires a positive integer.")
  }

  // Loading reserves the shared rhythm; resolved pages must release it so sparse
  // pagination reflects the rows that actually exist.
  const stableTableSurfaceMinHeight = !isLoading || stableSurfaceRowCount === undefined
    ? undefined
    : TABLE_HEADER_RHYTHM_HEIGHT_PX + (stableSurfaceRowCount * rowRhythm.rhythmHeight)

  const deleteConfirmation = actions?.deleteConfirmation

  const className = ui?.className
  const tableClassName = ui?.tableClassName

  const enableAutoColumnSizing = behavior?.enableAutoColumnSizing ?? false
  const expandColumnIds = behavior?.expandColumnIds
  const columnLayout = behavior?.columnLayout ?? "auto"
  const expandColumnIdSet = React.useMemo(() => new Set(expandColumnIds ?? []), [expandColumnIds])
  const legacyFillColumnId = expandColumnIds?.[0]
  const onRowClick = behavior?.onRowClick
  const onRowIntent = behavior?.onRowIntent
  const getRowActionLabel = behavior?.getRowActionLabel
  const [selectedBulkDeleteDialogOpen, setSelectedBulkDeleteDialogOpen] = React.useState(false)

  // Use table state hook
  const { table, columnSizeVars } = useTableState({
    data,
    columns,
    getRowId,
    pagination: externalPagination,
    setPagination: setExternalPagination,
    onPaginationChange,
    cursorPaginationSummary,
    paginationNavigation,
    paginationInfo,
    enableRowSelection,
    rowSelection: externalRowSelection,
    onRowSelectionChange: externalOnRowSelectionChange,
    sortingMode,
    sorting: externalSorting,
    onSortingChange: externalOnSortingChange,
    defaultSorting,
    columnVisibility: externalColumnVisibility,
    onColumnVisibilityChange: externalOnColumnVisibilityChange,
    enableAutoColumnSizing,
  })

  // Listen for selected row changes
  const rowSelection = table.getState().rowSelection
  const prevRowSelectionRef = React.useRef<Record<string, boolean>>({})
  React.useEffect(() => {
    if (onSelectionChange) {
      // Only call when rowSelection actually changes
      const prevSelection = prevRowSelectionRef.current
      const selectionChanged = Object.keys(rowSelection).length !== Object.keys(prevSelection).length ||
        Object.keys(rowSelection).some(key => rowSelection[key] !== prevSelection[key])

      if (selectionChanged) {
        prevRowSelectionRef.current = rowSelection
        const selectedRows = table.getFilteredSelectedRowModel().rows.map(row => row.original)
        onSelectionChange(selectedRows)
      }
    }
  }, [rowSelection, onSelectionChange, table])

  // Get selected row count
  const selectedCount = table.getFilteredSelectedRowModel().rows.length
  const externalSelectedRowsCount = externalSelectedRows?.length
  const prevExternalSelectedRowsCountRef = React.useRef(externalSelectedRowsCount)

  React.useEffect(() => {
    const prevExternalSelectedRowsCount = prevExternalSelectedRowsCountRef.current
    prevExternalSelectedRowsCountRef.current = externalSelectedRowsCount

    if (
      externalSelectedRows &&
      prevExternalSelectedRowsCount !== undefined &&
      prevExternalSelectedRowsCount > 0 &&
      externalSelectedRowsCount === 0 &&
      selectedCount > 0
    ) {
      table.resetRowSelection()
    }
  }, [externalSelectedRows, externalSelectedRowsCount, selectedCount, table])

  // Virtual scrolling setup
  const tableContainerRef = React.useRef<HTMLDivElement>(null)
  const [tableContainerWidth, setTableContainerWidth] = React.useState(0)
  const rows = table.getRowModel().rows
  const renderedLoadingRows = isLoading
    ? Array.from({ length: Math.max(0, Math.floor(loadingRowCount)) })
    : []

  // Enable virtual scrolling only for large datasets (>100 rows)
  const enableVirtualScrolling = !isLoading && rows.length > 100

  const rowVirtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => tableContainerRef.current,
    estimateSize: () => rowRhythm.estimatedHeight,
    overscan: 10, // Number of items to render outside visible area
    enabled: enableVirtualScrolling,
  })

  const virtualRows = enableVirtualScrolling ? rowVirtualizer.getVirtualItems() : []
  const totalSize = enableVirtualScrolling ? rowVirtualizer.getTotalSize() : 0

  const measureTableContainerWidth = React.useCallback(() => {
    const tableContainer = tableContainerRef.current
    if (!tableContainer) {
      return
    }

    const nextWidth = tableContainer.getBoundingClientRect().width
    setTableContainerWidth((currentWidth) => (
      Math.abs(currentWidth - nextWidth) > DATA_TABLE_RESIZE_OBSERVER_EPSILON_PX
        ? nextWidth
        : currentWidth
    ))
  }, [])

  React.useEffect(() => {
    const tableContainer = tableContainerRef.current
    if (!tableContainer || typeof ResizeObserver === "undefined") {
      return
    }

    const resizeObserver = new ResizeObserver(([entry]) => {
      const nextWidth = entry?.contentRect.width ?? tableContainer.getBoundingClientRect().width
      setTableContainerWidth((currentWidth) => (
        Math.abs(currentWidth - nextWidth) > DATA_TABLE_RESIZE_OBSERVER_EPSILON_PX
          ? nextWidth
          : currentWidth
      ))
    })

    resizeObserver.observe(tableContainer)

    return () => {
      resizeObserver.disconnect()
    }
  }, [])

  const tableColumnVisibility = table.getState().columnVisibility

  React.useLayoutEffect(() => {
    measureTableContainerWidth()
  }, [measureTableContainerWidth, tableColumnVisibility])

  const visibleLeafColumns = React.useMemo(() => {
    void columnSizeVars
    void tableColumnVisibility

    return table.getVisibleLeafColumns()
  }, [columnSizeVars, table, tableColumnVisibility])

  const tableWidthAllocation = React.useMemo(() => allocateSemanticColumnWidths(
    visibleLeafColumns.map((column) => {
      const minimum = column.columnDef.minSize ?? column.getSize()
      const legacyWidthPolicy = expandColumnIdSet.has(column.id)
        ? {
            mode: "flex" as const,
            flex: Math.max(column.getSize(), minimum, 1),
            fill: column.id === legacyFillColumnId,
          }
        : undefined

      return {
        id: column.id,
        size: column.getSize(),
        minSize: minimum,
        maxSize: column.columnDef.maxSize,
        widthPolicy: column.columnDef.meta?.widthPolicy ?? legacyWidthPolicy,
      }
    }),
    tableContainerWidth
  ), [expandColumnIdSet, legacyFillColumnId, tableContainerWidth, visibleLeafColumns])
  const tableMinWidth = tableWidthAllocation.minimumWidth
  const tableColumnWidthStyles = React.useMemo(() => (
    visibleLeafColumns.map((column) => `${tableWidthAllocation.widthsById.get(column.id) ?? column.getSize()}px`)
  ), [tableWidthAllocation.widthsById, visibleLeafColumns])

  const getHeaderWidthStyle = React.useCallback((headerId: string, columnId: string): React.CSSProperties => {
    const width = tableWidthAllocation.widthsById.get(columnId)
    const resolvedWidth = width === undefined
      ? `calc(var(--header-${headerId}-size) * 1px)`
      : `${width}px`

    return { width: resolvedWidth, minWidth: resolvedWidth }
  }, [tableWidthAllocation.widthsById])

  const getCellWidthStyle = React.useCallback((columnId: string): React.CSSProperties => {
    const width = tableWidthAllocation.widthsById.get(columnId)
    const resolvedWidth = width === undefined
      ? `calc(var(--col-${columnId}-size) * 1px)`
      : `${width}px`

    return { width: resolvedWidth, minWidth: resolvedWidth }
  }, [tableWidthAllocation.widthsById])

  const getStickyColumnClassName = React.useCallback(
    (stickyRight: boolean | undefined, slot: "header" | "cell") => (
      getDataTableStickyColumnShellClassName(stickyRight, slot)
    ),
    []
  )

  const handleRowClick = React.useCallback((event: React.MouseEvent, row: (typeof rows)[number]) => {
    if (!onRowClick || shouldIgnoreRowActivation(event)) {
      return
    }

    onRowClick(row.original)
  }, [onRowClick])

  const handleRowIntent = React.useCallback((row: (typeof rows)[number]) => {
    onRowIntent?.(row.original)
  }, [onRowIntent])

  const handleRowKeyDown = React.useCallback((event: React.KeyboardEvent, row: (typeof rows)[number]) => {
    if (!onRowClick || shouldIgnoreRowActivation(event)) {
      return
    }

    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault()
      onRowClick(row.original)
    }
  }, [onRowClick])

  const handleClearSelection = React.useCallback(() => {
    table.resetRowSelection()
  }, [table])

  const handleSelectedBulkDeleteClick = React.useCallback(() => {
    if (!onBulkDelete || selectedCount === 0) {
      return
    }

    if (deleteConfirmation) {
      setSelectedBulkDeleteDialogOpen(true)
      return
    }

    onBulkDelete()
  }, [deleteConfirmation, onBulkDelete, selectedCount])

  const handleSelectedBulkDeleteConfirm = React.useCallback(() => {
    setSelectedBulkDeleteDialogOpen(false)
    onBulkDelete?.()
  }, [onBulkDelete])

  const selectedRowActions: SelectedRowActionBarAction[] = [
    ...configuredSelectedRowActions,
    ...(showBulkDelete && onBulkDelete
      ? [{
          key: "delete",
          label: bulkDeleteLabel,
          icon: IconTrash,
          tone: "destructive" as const,
          group: "danger",
          onClick: handleSelectedBulkDeleteClick,
        }]
      : []),
  ]

  const getLoadingRowHeight = React.useCallback((rowIndex: number) => {
    if (typeof loadingRowHeightEstimate === "function") {
      return loadingRowHeightEstimate(rowIndex)
    }

    return loadingRowHeightEstimate
  }, [loadingRowHeightEstimate])

  const loadingRows = isLoading
    ? renderedLoadingRows.map((_, rowIndex) => (
        <TableRow
          key={`loading-${rowIndex}`}
          aria-hidden="true"
          className={cn(rowRhythm.rowClassName, "pointer-events-none hover:bg-card")}
          style={{ height: getLoadingRowHeight(rowIndex) }}
        >
          {visibleLeafColumns.map((column, columnIndex) => (
            <TableCell
              key={column.id}
              className={cn(
                rowRhythm.cellClassName,
                getStickyColumnClassName(column.columnDef.meta?.stickyRight, "cell")
              )}
              style={getCellWidthStyle(column.id)}
            >
              <Skeleton
                className={cn(
                  "h-4 rounded-full",
                  columnIndex === 0 && "w-4",
                  columnIndex === 1 && "w-32",
                  columnIndex > 1 && columnIndex < visibleLeafColumns.length - 1 && "w-24",
                  columnIndex === visibleLeafColumns.length - 1 && "ml-auto size-8 rounded-md"
                )}
              />
            </TableCell>
          ))}
        </TableRow>
      ))
    : null

  const initialLoadingToolbarActionCount = [
    showColumnVisibility,
    Boolean(toolbarRight),
    Boolean(exportOptions?.length),
    Boolean(showRefresh && onRefresh),
    Boolean(showAddButton && onAddNew),
    Boolean(showBulkAdd && onBulkAdd),
  ].filter(Boolean).length

  return (
    <div
      data-slot="unified-data-table"
      data-loading-controlled-table-skeleton-variant={isInitialLoading ? "initial" : undefined}
      data-loading-presentation={isLoading ? loadingPresentation : undefined}
      aria-busy={isLoading || undefined}
      className={cn("w-full min-w-0 space-y-4", className)}
    >
      {/* Toolbar */}
      {!hideToolbar && (
        <div {...getLoadingStructureSlotAttributes(loadingSlots.toolbar)}>
          {isInitialLoading ? (
            <InitialDataTableChromePlaceholder
              filterCount={initialLoadingToolbarFilterCount}
              actionCount={initialLoadingToolbarActionCount}
              toolbarDensity={toolbarDensity}
            />
          ) : (
            <TableToolbar
              table={table}
              searchMode={searchMode}
              searchPlaceholder={searchPlaceholder}
              searchValue={searchValue}
              onSearch={onSearch}
              isSearching={isSearching}
              filterFields={filterFields}
              filterExamples={filterExamples}
              toolbarLeft={toolbarLeft}
              toolbarRight={toolbarRight}
              showColumnVisibility={showColumnVisibility}
              exportOptions={exportOptions}
              selectedCount={selectedCount}
              toolbarDensity={toolbarDensity}
            >
              <TableActions
                selectedCount={selectedCount}
                onAddNew={onAddNew}
                onAddHover={onAddHover}
                addButtonLabel={addButtonLabel}
                showAddButton={showAddButton}
                onBulkAdd={onBulkAdd}
                bulkAddLabel={bulkAddLabel}
                showBulkAdd={showBulkAdd}
                onRefresh={onRefresh}
                refreshLabel={refreshLabel}
                showRefresh={showRefresh}
                isRefreshing={isRefreshing}
                toolbarDensity={toolbarDensity}
              />
            </TableToolbar>
          )}
        </div>
      )}

      {/* The bordered surface owns the table only; pagination stays in page flow. */}
      <div
        ref={tableContainerRef}
        data-slot="data-table"
        data-table-variant="shell"
        data-row-rhythm={rowDensity}
        data-column-layout={columnLayout}
        data-stable-surface-row-count={stableSurfaceRowCount}
        className={cn(
          "rounded-md border border-border bg-card overflow-x-auto",
          enableVirtualScrolling && DATA_TABLE_VIRTUAL_SCROLL_CLASSNAME,
          tableClassName
        )}
        {...getLoadingStructureSlotAttributes(loadingSlots.body)}
        style={stableTableSurfaceMinHeight === undefined
          ? undefined
          : { minHeight: `${stableTableSurfaceMinHeight}px` }}
      >
        <table
          className={cn("caption-bottom text-sm w-full", columnLayout === "fixed" && "table-fixed")}
          style={{
            ...columnSizeVars,
            minWidth: tableMinWidth,
          }}
        >
          {columnLayout === "fixed" && (
            <colgroup>
              {tableColumnWidthStyles.map((width, index) => (
                <col key={visibleLeafColumns[index]?.id ?? index} style={{ width }} />
              ))}
            </colgroup>
          )}
          <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <TableHead
                      key={header.id}
                      colSpan={header.colSpan}
                      style={getHeaderWidthStyle(header.id, header.column.id)}
                      className={cn(
                        "group relative",
                        getStickyColumnClassName(header.column.columnDef.meta?.stickyRight, "header")
                      )}
                    >
                      {header.isPlaceholder ? null : isInitialLoading ? (
                        <Skeleton
                          className={cn(
                            "h-4 rounded-full",
                            header.index === 0 ? "w-4" : header.index === 1 ? "w-24" : "w-16"
                          )}
                        />
                      ) : flexRender(header.column.columnDef.header, header.getContext())}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
          </TableHeader>
          <TableBody
            style={enableVirtualScrolling ? {
              height: `${totalSize}px`,
              position: 'relative',
            } : undefined}
          >
              {isLoading ? (
                loadingRows
              ) : rows.length ? (
                enableVirtualScrolling ? (
                  // Virtual scrolling mode for large datasets
                  virtualRows.map((virtualRow) => {
                    const row = rows[virtualRow.index]
                    return (
                      <TableRow
                        key={row.id}
                        data-state={row.getIsSelected() && "selected"}
                        tabIndex={onRowClick ? 0 : undefined}
                        aria-label={getRowActionLabel?.(row.original)}
                        className={cn(
                          rowRhythm.rowClassName,
                          "group",
                          onRowClick && "cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 focus-visible:ring-inset"
                        )}
                        onClick={(event) => handleRowClick(event, row)}
                        onPointerEnter={() => handleRowIntent(row)}
                        onFocus={() => handleRowIntent(row)}
                        onKeyDown={(event) => handleRowKeyDown(event, row)}
                        style={{
                          position: 'absolute',
                          top: 0,
                          left: 0,
                          width: '100%',
                          transform: `translateY(${virtualRow.start}px)`,
                        }}
                      >
                        {row.getVisibleCells().map((cell) => (
                          <TableCell
                            key={cell.id}
                            className={cn(
                              rowRhythm.cellClassName,
                              getStickyColumnClassName(cell.column.columnDef.meta?.stickyRight, "cell")
                            )}
                            style={getCellWidthStyle(cell.column.id)}
                          >
                            {flexRender(cell.column.columnDef.cell, cell.getContext())}
                          </TableCell>
                        ))}
                      </TableRow>
                    )
                  })
                ) : (
                  // Standard rendering for small datasets
                  rows.map((row) => (
                    <TableRow
                      key={row.id}
                      data-state={row.getIsSelected() && "selected"}
                      tabIndex={onRowClick ? 0 : undefined}
                      aria-label={getRowActionLabel?.(row.original)}
                      className={cn(
                        rowRhythm.rowClassName,
                        "group",
                        onRowClick && "cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 focus-visible:ring-inset"
                      )}
                      onClick={(event) => handleRowClick(event, row)}
                      onPointerEnter={() => handleRowIntent(row)}
                      onFocus={() => handleRowIntent(row)}
                      onKeyDown={(event) => handleRowKeyDown(event, row)}
                    >
                      {row.getVisibleCells().map((cell) => (
                        <TableCell
                          key={cell.id}
                          className={cn(
                            rowRhythm.cellClassName,
                            getStickyColumnClassName(cell.column.columnDef.meta?.stickyRight, "cell")
                          )}
                          style={getCellWidthStyle(cell.column.id)}
                        >
                          {flexRender(cell.column.columnDef.cell, cell.getContext())}
                        </TableCell>
                      ))}
                    </TableRow>
                  ))
                )
              ) : (
                <TableRow>
                  <TableCell colSpan={columns.length} className="h-24 text-center">
                    {emptyComponent || emptyMessage}
                  </TableCell>
                </TableRow>
              )}
          </TableBody>
        </table>
      </div>
      {!hidePagination && (
        <div
          data-slot="data-table-pagination-surface"
          {...getLoadingStructureSlotAttributes(loadingSlots.pagination)}
        >
          {isInitialLoading ? (
            <CompactPaginationSkeleton
              mode={paginationNavigation?.mode === "cursor" ? "cursor" : "numbered"}
              showSummary={paginationNavigation?.mode !== "cursor" || Boolean(cursorPaginationSummary)}
              buttonCount={paginationNavigation?.mode === "cursor" ? 3 : 4}
            />
          ) : (
            <DataTablePagination
              table={table}
              paginationInfo={paginationInfo}
              cursorPaginationSummary={cursorPaginationSummary}
              paginationNavigation={paginationNavigation}
              pageSizeOptions={pageSizeOptions}
            />
          )}
        </div>
      )}
      <SelectedRowActionBar
        selectedCount={selectedCount}
        ariaLabel={tDataTable("selected", { count: selectedCount })}
        countLabel={tDataTable("selected", { count: selectedCount })}
        actions={selectedRowActions}
        onClearSelection={handleClearSelection}
        clearSelectionLabel={tDataTable("deselectAll")}
      />
      {deleteConfirmation && (
        <AlertDialog
          open={selectedBulkDeleteDialogOpen}
          onOpenChange={setSelectedBulkDeleteDialogOpen}
        >
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>
                {deleteConfirmation.title || "Confirm Delete"}
              </AlertDialogTitle>
              <AlertDialogDescription>
                {typeof deleteConfirmation.description === "function"
                  ? deleteConfirmation.description(selectedCount)
                  : deleteConfirmation.description ||
                    `Are you sure you want to delete ${selectedCount} selected item(s)? This action cannot be undone.`}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogClose variant="outline">
                {deleteConfirmation.cancelLabel || "Cancel"}
              </AlertDialogClose>
              <AlertDialogClose
                onClick={handleSelectedBulkDeleteConfirm}
                className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              >
                {deleteConfirmation.confirmLabel || "Delete"}
              </AlertDialogClose>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      )}
    </div>
  )
}
