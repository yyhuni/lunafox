"use client"

import * as React from "react"
import { useLocale } from "next-intl"
import {
  ColumnFiltersState,
  ColumnSizingState,
  SortingState,
  VisibilityState,
  Updater,
  useReactTable,
  getCoreRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  type ColumnDef,
} from "@tanstack/react-table"
import { calculateColumnWidths } from "@/lib/table-utils"
import type { PaginationState } from "@/types/data-table.types"

function hasColumnSizingChanged(prev: ColumnSizingState, next: ColumnSizingState): boolean {
  const keys = new Set([...Object.keys(prev), ...Object.keys(next)])
  for (const key of keys) {
    if (prev[key] !== next[key]) {
      return true
    }
  }
  return false
}

export interface UseTableStateProps<TData> {
  data: TData[]
  columns: ColumnDef<TData, unknown>[]
  getRowId?: (row: TData, index: number) => string

  // Pagination
  pagination?: PaginationState
  setPagination?: (pagination: PaginationState) => void
  onPaginationChange?: (pagination: PaginationState) => void
  paginationInfo?: {
    totalPages: number
    total: number
  }

  // Selection
  enableRowSelection?: boolean
  rowSelection?: Record<string, boolean>
  onRowSelectionChange?: (selection: Record<string, boolean>) => void

  // Sorting
  sorting?: SortingState
  onSortingChange?: (sorting: SortingState) => void
  defaultSorting?: SortingState

  // Column visibility
  columnVisibility?: VisibilityState
  onColumnVisibilityChange?: (visibility: VisibilityState) => void

  // Auto column sizing
  enableAutoColumnSizing?: boolean
}

export function useTableState<TData>({
  data,
  columns,
  getRowId,
  pagination: externalPagination,
  setPagination: setExternalPagination,
  onPaginationChange,
  paginationInfo,
  enableRowSelection = true,
  rowSelection: externalRowSelection,
  onRowSelectionChange: externalOnRowSelectionChange,
  sorting: externalSorting,
  onSortingChange: externalOnSortingChange,
  defaultSorting = [],
  columnVisibility: externalColumnVisibility,
  onColumnVisibilityChange: externalOnColumnVisibilityChange,
  enableAutoColumnSizing = false,
}: UseTableStateProps<TData>) {
  const locale = useLocale()

  // Internal state
  const [internalRowSelection, setInternalRowSelection] = React.useState<Record<string, boolean>>({})
  const [internalColumnVisibility, setInternalColumnVisibility] = React.useState<VisibilityState>({})
  const [internalSorting, setInternalSorting] = React.useState<SortingState>(defaultSorting)
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>([])
  const [columnSizing, setColumnSizing] = React.useState<ColumnSizingState>({})
  const [internalPagination, setInternalPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })
  const lastAutoSizingKeyRef = React.useRef("")
  const autoSizingPausedByUserRef = React.useRef(false)

  // Use external state or internal state
  const rowSelection = externalRowSelection ?? internalRowSelection
  const columnVisibility = externalColumnVisibility ?? internalColumnVisibility
  const sorting = externalSorting ?? internalSorting

  // Determine whether to use external pagination control
  const isExternalPagination = !!(externalPagination && (onPaginationChange || setExternalPagination))
  const pagination = externalPagination ?? internalPagination

  // Use ref to store the latest pagination value to avoid closure issues
  const paginationRef = React.useRef(pagination)
  paginationRef.current = pagination

  // Pagination update handler
  const handlePaginationChange = React.useCallback((updater: Updater<PaginationState>) => {
    const currentPagination = paginationRef.current
    const newPagination = typeof updater === 'function' ? updater(currentPagination) : updater

    // No change in value, don't update
    if (newPagination.pageIndex === currentPagination.pageIndex &&
        newPagination.pageSize === currentPagination.pageSize) {
      return
    }

    if (isExternalPagination) {
      // External pagination control
      if (onPaginationChange) {
        onPaginationChange(newPagination)
      } else if (setExternalPagination) {
        setExternalPagination(newPagination)
      }
    } else {
      // Internal pagination control
      setInternalPagination(newPagination)
    }
  }, [isExternalPagination, onPaginationChange, setExternalPagination])

  // Handle state updates (supports Updater pattern)
  const handleRowSelectionChange = React.useCallback((updater: Updater<Record<string, boolean>>) => {
    const newValue = typeof updater === 'function' ? updater(rowSelection) : updater
    if (externalOnRowSelectionChange) {
      externalOnRowSelectionChange(newValue)
    } else {
      setInternalRowSelection(newValue)
    }
  }, [rowSelection, externalOnRowSelectionChange])

  const handleSortingChange = React.useCallback((updater: Updater<SortingState>) => {
    const newValue = typeof updater === 'function' ? updater(sorting) : updater
    if (externalOnSortingChange) {
      externalOnSortingChange(newValue)
    } else {
      setInternalSorting(newValue)
    }
  }, [sorting, externalOnSortingChange])

  const handleColumnVisibilityChange = React.useCallback((updater: Updater<VisibilityState>) => {
    const newValue = typeof updater === 'function' ? updater(columnVisibility) : updater
    if (externalOnColumnVisibilityChange) {
      externalOnColumnVisibilityChange(newValue)
    } else {
      setInternalColumnVisibility(newValue)
    }
  }, [columnVisibility, externalOnColumnVisibilityChange])

  const handleColumnSizingChange = React.useCallback((updater: Updater<ColumnSizingState>) => {
    setColumnSizing((prev) => {
      const next = typeof updater === "function" ? updater(prev) : updater

      if (enableAutoColumnSizing && hasColumnSizingChanged(prev, next)) {
        autoSizingPausedByUserRef.current = true
      }

      return next
    })
  }, [enableAutoColumnSizing])

  const resolvedGetRowId = React.useCallback((row: TData, index: number, parent?: { id: string }) => {
    const externalId = getRowId?.(row, index)
    if (typeof externalId === "string" && externalId.trim() !== "") {
      return externalId
    }

    const rowWithFallbackId = row as { id?: string | number; _id?: string | number }
    const fallbackId = rowWithFallbackId.id ?? rowWithFallbackId._id
    if (fallbackId !== undefined && fallbackId !== null && String(fallbackId).trim() !== "") {
      return String(fallbackId)
    }

    return parent ? `${parent.id}.${index}` : `row-${index}`
  }, [getRowId])

  // Filter valid data (only remove nullish rows)
  const validData = React.useMemo(() => {
    return (data || []).filter((item): item is TData => item != null)
  }, [data])

  const autoSizingKey = React.useMemo(() => {
    if (!enableAutoColumnSizing || validData.length === 0) return ""

    const columnIds = columns
      .map((col) => {
        const colDef = col as { accessorKey?: string; id?: string }
        return colDef.accessorKey || colDef.id || ""
      })
      .filter((id): id is string => id.length > 0)

    if (columnIds.length === 0) return ""

    const sample = validData.slice(0, 20).map((row) => {
      const record = row as Record<string, unknown>
      return columnIds
        .map((id) => {
          const value = record[id]
          if (value === null || value === undefined) return ""
          if (typeof value === "object") return "[object]"
          return String(value)
        })
        .join("|")
    }).join("||")

    return `${locale}:${validData.length}:${columnIds.join(",")}:${sample}`
  }, [columns, enableAutoColumnSizing, locale, validData])

  // Auto column sizing: calculate optimal widths based on content
  React.useEffect(() => {
    if (!enableAutoColumnSizing || validData.length === 0) {
      return
    }
    // If the user manually resizes columns, keep their choice and stop auto recalculation.
    if (autoSizingPausedByUserRef.current) {
      return
    }
    if (autoSizingKey && autoSizingKey === lastAutoSizingKeyRef.current) {
      return
    }

    // Build header labels from column meta
    const headerLabels: Record<string, string> = {}
    for (const col of columns) {
      const colDef = col as { accessorKey?: string; id?: string; meta?: { title?: string } }
      const colId = colDef.accessorKey || colDef.id
      if (colId && colDef.meta?.title) {
        headerLabels[colId] = colDef.meta.title
      }
    }

    const calculatedWidths = calculateColumnWidths({
      data: validData as Record<string, unknown>[],
      columns: columns as Array<{
        accessorKey?: string
        id?: string
        size?: number
        minSize?: number
        maxSize?: number
      }>,
      headerLabels,
      locale,
    })

    lastAutoSizingKeyRef.current = autoSizingKey
    if (Object.keys(calculatedWidths).length > 0) {
      setColumnSizing((prev) => {
        let changed = false
        const next: ColumnSizingState = { ...prev }

        for (const [key, value] of Object.entries(calculatedWidths)) {
          if (next[key] !== value) {
            next[key] = value
            changed = true
          }
        }

        return changed ? next : prev
      })
    }
  }, [autoSizingKey, enableAutoColumnSizing, validData, columns, locale])

  // Create table instance
  const table = useReactTable({
    data: validData,
    columns,
    state: {
      sorting,
      columnVisibility,
      rowSelection,
      columnFilters,
      pagination,
      columnSizing,
    },
    // Column resizing configuration - following TanStack Table official recommendations
    enableColumnResizing: true,
    columnResizeMode: 'onChange',
    onColumnSizingChange: handleColumnSizingChange,
    // Default column configuration
    defaultColumn: {
      minSize: 50,
      maxSize: 1000,
    },
    pageCount: paginationInfo?.totalPages ?? -1,
    manualPagination: !!paginationInfo,
    getRowId: resolvedGetRowId,
    enableRowSelection,
    onRowSelectionChange: handleRowSelectionChange,
    onSortingChange: handleSortingChange,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: handleColumnVisibilityChange,
    onPaginationChange: handlePaginationChange,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
  })

  /**
   * Following TanStack Table's official high-performance approach:
   * Calculate all column widths once at the table root element, store as CSS variables
   * Avoid calling column.getSize() on every cell
   */
  const { columnSizingInfo, columnSizing: tableColumnSizing } = table.getState()

  const columnSizeVars = React.useMemo(() => {
    void columnSizingInfo
    void tableColumnSizing
    const headers = table.getFlatHeaders()
    const colSizes: Record<string, number> = {}
    for (let i = 0; i < headers.length; i++) {
      const header = headers[i]!
      colSizes[`--header-${header.id}-size`] = header.getSize()
      colSizes[`--col-${header.column.id}-size`] = header.column.getSize()
    }
    return colSizes
  }, [columnSizingInfo, tableColumnSizing, table])

  return {
    table,
    columnSizeVars,
  }
}
