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
import { calculateColumnWidths, getBadgeMinWidthPx, getColumnHeaderMinWidthPx } from "@/lib/table-utils"
import type {
  CursorPaginationNavigation,
  CursorPaginationSummary,
  DataTableSortingMode,
  PaginationState,
} from "@/types/data-table.types"

export interface UseTableStateProps<TData> {
  data: TData[]
  columns: ColumnDef<TData, unknown>[]
  getRowId?: (row: TData, index: number) => string

  // Pagination
  pagination?: PaginationState
  setPagination?: (pagination: PaginationState) => void
  onPaginationChange?: (pagination: PaginationState) => void
  cursorPaginationSummary?: CursorPaginationSummary
  paginationNavigation?: CursorPaginationNavigation
  paginationInfo?: {
    totalPages: number
    total: number
  }

  // Selection
  enableRowSelection?: boolean
  rowSelection?: Record<string, boolean>
  onRowSelectionChange?: (selection: Record<string, boolean>) => void

  // Sorting
  sortingMode?: DataTableSortingMode
  sorting?: SortingState
  onSortingChange?: (sorting: SortingState) => void
  defaultSorting?: SortingState

  // Column visibility
  columnVisibility?: VisibilityState
  onColumnVisibilityChange?: (visibility: VisibilityState) => void

  // Auto column sizing
  enableAutoColumnSizing?: boolean
}

function getSingleBadgeFloorPx<TData>(
  columnDef: ColumnDef<TData, unknown> & {
    accessorKey?: string
    id?: string
    meta?: {
      singleBadge?: boolean
      singleBadgeValue?: (row: TData) => string | null | undefined
      singleBadgeValues?: readonly string[]
    }
  },
  rows: TData[]
): number {
  if (!columnDef.meta?.singleBadge) {
    return 0
  }

  const columnId = columnDef.accessorKey || columnDef.id
  const badgeValues = new Set(columnDef.meta.singleBadgeValues ?? [])

  let maxBadgeWidth = 0

  for (const row of rows) {
    const resolvedValue = columnDef.meta.singleBadgeValue
      ? columnDef.meta.singleBadgeValue(row)
      : columnId
        ? (() => {
          const record = row as Record<string, unknown>
          const rawValue = record[columnId]
          if (rawValue === null || rawValue === undefined || rawValue === "") {
            return null
          }
          if (typeof rawValue === "string" || typeof rawValue === "number" || typeof rawValue === "boolean") {
            return String(rawValue)
          }
          return null
        })()
        : null

    if (resolvedValue) {
      badgeValues.add(resolvedValue)
    }
  }

  for (const value of badgeValues) {
    if (value) {
      maxBadgeWidth = Math.max(maxBadgeWidth, getBadgeMinWidthPx(value))
    }
  }

  return maxBadgeWidth
}

export function useTableState<TData>({
  data,
  columns,
  getRowId,
  pagination: externalPagination,
  setPagination: setExternalPagination,
  onPaginationChange,
  cursorPaginationSummary,
  paginationNavigation,
  paginationInfo,
  enableRowSelection = true,
  rowSelection: externalRowSelection,
  onRowSelectionChange: externalOnRowSelectionChange,
  sortingMode,
  sorting: externalSorting,
  onSortingChange: externalOnSortingChange,
  defaultSorting = [],
  columnVisibility: externalColumnVisibility,
  onColumnVisibilityChange: externalOnColumnVisibilityChange,
  enableAutoColumnSizing = false,
}: UseTableStateProps<TData>) {
  const locale = useLocale()
  const resolvedSortingMode = sortingMode ?? (paginationInfo ? "none" : "client")

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

  const normalizedColumns = React.useMemo(() => {
    return columns.map((column) => {
      const columnDef = column as ColumnDef<TData, unknown> & {
        accessorKey?: string
        id?: string
        minSize?: number
        size?: number
        maxSize?: number
        meta?: {
          title?: string
          singleBadge?: boolean
          singleBadgeValue?: (row: TData) => string | null | undefined
          singleBadgeValues?: readonly string[]
          enableAutoSize?: boolean
          orderBy?: string
          serverSortPerformance?: string
          firstSortDirection?: "asc" | "desc"
        }
        enableSorting?: boolean
      }

      const title = columnDef.meta?.title
      const canSortColumn = columnDef.enableSorting !== false && (
        resolvedSortingMode !== "server" || Boolean(columnDef.meta?.orderBy && columnDef.meta?.serverSortPerformance)
      )
      if (!title) {
        return {
          ...columnDef,
          enableSorting: canSortColumn,
        } satisfies ColumnDef<TData, unknown>
      }

      const declaredMinSize = columnDef.minSize ?? 50
      const declaredSize = columnDef.size
      const declaredMaxSize = columnDef.maxSize
      const headerMinWidthPx = getColumnHeaderMinWidthPx(title, {
        includeSortIcon: resolvedSortingMode !== "none" && canSortColumn,
      })
      const singleBadgeFloorPx = getSingleBadgeFloorPx(columnDef, validData.slice(0, 50))
      const normalizedMinSize = Math.max(declaredMinSize, headerMinWidthPx, singleBadgeFloorPx)

      return {
        ...columnDef,
        enableSorting: canSortColumn,
        size: declaredSize === undefined ? declaredSize : Math.max(declaredSize, normalizedMinSize),
        minSize: normalizedMinSize,
        maxSize: declaredMaxSize === undefined ? declaredMaxSize : Math.max(declaredMaxSize, normalizedMinSize),
      } satisfies ColumnDef<TData, unknown>
    })
  }, [columns, resolvedSortingMode, validData])

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
      columns: normalizedColumns as Array<{
        accessorKey?: string
        id?: string
        size?: number
        minSize?: number
        maxSize?: number
        enableSorting?: boolean
        meta?: { enableAutoSize?: boolean }
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
  }, [autoSizingKey, enableAutoColumnSizing, validData, columns, locale, normalizedColumns])

  // Create table instance
  const table = useReactTable({
    data: validData,
    columns: normalizedColumns,
    state: {
      sorting,
      columnVisibility,
      rowSelection,
      columnFilters,
      pagination,
      columnSizing,
    },
    // Default column configuration
    defaultColumn: {
      minSize: 50,
      maxSize: 1000,
    },
    // Cursor-backed lists have no reliable random-access page count. Keep the
    // table boundary open and let explicit token availability own navigation.
    pageCount: paginationNavigation?.mode === "cursor"
      ? -1
      : paginationInfo?.totalPages ?? -1,
    manualPagination: !!paginationInfo || !!cursorPaginationSummary,
    getRowId: resolvedGetRowId,
    enableSorting: resolvedSortingMode !== "none",
    manualSorting: resolvedSortingMode === "server",
    enableRowSelection,
    onRowSelectionChange: handleRowSelectionChange,
    onSortingChange: handleSortingChange,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: handleColumnVisibilityChange,
    onPaginationChange: handlePaginationChange,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: resolvedSortingMode === "client" ? getSortedRowModel() : undefined,
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
