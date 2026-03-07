"use client"

import * as React from "react"
import {
  flexRender,
} from "@tanstack/react-table"
import { useVirtualizer } from "@tanstack/react-virtual"

import {
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { cn } from "@/lib/utils"

import { DataTablePagination } from "./pagination"
import { ColumnResizer } from "./column-resizer"
import { useTableState } from "./use-table-state"
import { TableToolbar } from "./table-toolbar"
import { TableActions } from "./table-actions"
import type {
  UnifiedDataTableProps,
} from "@/types/data-table.types"

/**
 * Unified data table component
 *
 * Features:
 * - Generic support, type safety
 * - Row selection, sorting, column visibility, column resizing
 * - Client/server-side pagination
 * - Simple search/smart filtering
 * - Bulk operations, download functionality
 * - Confirmation dialogs
 * - Virtual scrolling for large datasets (10,000+ rows)
 */
export function UnifiedDataTable<TData>(props: UnifiedDataTableProps<TData>) {
  const { data, columns, getRowId, state, ui, behavior, actions } = props

  const externalPagination = state?.pagination
  const setExternalPagination = state?.setPagination
  const paginationInfo = state?.paginationInfo
  const onPaginationChange = state?.onPaginationChange
  const hidePagination = ui?.hidePagination ?? false
  const pageSizeOptions = ui?.pageSizeOptions

  const hideToolbar = ui?.hideToolbar ?? false
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

  const onBulkDelete = actions?.onBulkDelete
  const bulkDeleteLabel = actions?.bulkDeleteLabel ?? "Delete"
  const showBulkDelete = actions?.showBulkDelete ?? true

  const onAddNew = actions?.onAddNew
  const onAddHover = actions?.onAddHover
  const addButtonLabel = actions?.addButtonLabel ?? "Add"
  const showAddButton = actions?.showAddButton ?? true

  const onBulkAdd = actions?.onBulkAdd
  const bulkAddLabel = actions?.bulkAddLabel ?? "Bulk Add"
  const showBulkAdd = actions?.showBulkAdd ?? true

  const downloadOptions = actions?.downloadOptions

  const externalColumnVisibility = state?.columnVisibility
  const externalOnColumnVisibilityChange = state?.onColumnVisibilityChange

  const externalSorting = state?.sorting
  const externalOnSortingChange = state?.onSortingChange
  const defaultSorting = state?.defaultSorting ?? []

  const emptyMessage = ui?.emptyMessage ?? "No results"
  const emptyComponent = ui?.emptyComponent

  const deleteConfirmation = actions?.deleteConfirmation

  const className = ui?.className
  const tableClassName = ui?.tableClassName

  const enableAutoColumnSizing = behavior?.enableAutoColumnSizing ?? false
  const expandColumnIds = behavior?.expandColumnIds
  const expandColumnIdSet = React.useMemo(() => new Set(expandColumnIds ?? []), [expandColumnIds])

  // Use table state hook
  const { table, columnSizeVars } = useTableState({
    data,
    columns,
    getRowId,
    pagination: externalPagination,
    setPagination: setExternalPagination,
    onPaginationChange,
    paginationInfo,
    enableRowSelection,
    rowSelection: externalRowSelection,
    onRowSelectionChange: externalOnRowSelectionChange,
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

  // Virtual scrolling setup
  const tableContainerRef = React.useRef<HTMLDivElement>(null)
  const rows = table.getRowModel().rows

  // Enable virtual scrolling only for large datasets (>100 rows)
  const enableVirtualScrolling = rows.length > 100

  const rowVirtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => tableContainerRef.current,
    estimateSize: () => 53, // Estimated row height in pixels
    overscan: 10, // Number of items to render outside visible area
    enabled: enableVirtualScrolling,
  })

  const virtualRows = enableVirtualScrolling ? rowVirtualizer.getVirtualItems() : []
  const totalSize = enableVirtualScrolling ? rowVirtualizer.getTotalSize() : 0

  const getHeaderWidthStyle = React.useCallback((headerId: string, columnId: string): React.CSSProperties => {
    const minWidth = `calc(var(--header-${headerId}-size) * 1px)`
    if (expandColumnIdSet.has(columnId)) {
      return { minWidth }
    }
    return {
      width: minWidth,
      minWidth,
    }
  }, [expandColumnIdSet])

  const getCellWidthStyle = React.useCallback((columnId: string): React.CSSProperties => {
    const minWidth = `calc(var(--col-${columnId}-size) * 1px)`
    if (expandColumnIdSet.has(columnId)) {
      return { minWidth }
    }
    return {
      width: minWidth,
      minWidth,
    }
  }, [expandColumnIdSet])

  return (
    <div className={cn("w-full space-y-4", className)}>
      {/* Toolbar */}
      {!hideToolbar && (
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
          downloadOptions={downloadOptions}
          selectedCount={selectedCount}
        >
          <TableActions
            selectedCount={selectedCount}
            onBulkDelete={onBulkDelete}
            bulkDeleteLabel={bulkDeleteLabel}
            showBulkDelete={showBulkDelete}
            deleteConfirmation={deleteConfirmation}
            onAddNew={onAddNew}
            onAddHover={onAddHover}
            addButtonLabel={addButtonLabel}
            showAddButton={showAddButton}
            onBulkAdd={onBulkAdd}
            bulkAddLabel={bulkAddLabel}
            showBulkAdd={showBulkAdd}
          />
        </TableToolbar>
      )}

      {/* Table - Following TanStack Table official recommendations using CSS variables */}
      <div
        ref={tableContainerRef}
        data-slot="data-table"
        className={cn(
          "rounded-md border overflow-x-auto",
          enableVirtualScrolling && "max-h-[600px] overflow-y-auto",
          tableClassName
        )}
      >
        <table
          className="w-full caption-bottom text-sm"
          style={{
            ...columnSizeVars,
            minWidth: table.getTotalSize(),
          }}
        >
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <TableHead
                    key={header.id}
                    colSpan={header.colSpan}
                    style={getHeaderWidthStyle(header.id, header.column.id)}
                    className="relative group"
                  >
                    {header.isPlaceholder
                      ? null
                      : flexRender(header.column.columnDef.header, header.getContext())}
                    <ColumnResizer header={header} />
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
            {rows.length ? (
              enableVirtualScrolling ? (
                // Virtual scrolling mode for large datasets
                virtualRows.map((virtualRow) => {
                  const row = rows[virtualRow.index]
                  return (
                    <TableRow
                      key={row.id}
                      data-state={row.getIsSelected() && "selected"}
                      className="group"
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
                    className="group"
                  >
                    {row.getVisibleCells().map((cell) => (
                      <TableCell
                        key={cell.id}
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

      {/* Pagination */}
      {!hidePagination && (
        <DataTablePagination
          table={table}
          paginationInfo={paginationInfo}
          pageSizeOptions={pageSizeOptions}
        />
      )}
    </div>
  )
}
