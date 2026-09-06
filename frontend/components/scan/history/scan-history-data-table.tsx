"use client"

import * as React from "react"
import type { ColumnDef, SortingState } from "@tanstack/react-table"

import { BusinessListDataTable } from "@/components/shared/data-table"
import type { SelectedRowActionBarAction } from "@/components/shared/data-table"

import type { ScanRecord } from "@/types/scan.types"
import type { PaginationInfo } from "@/types/common.types"
import type {
  CursorPaginationNavigation,
  CursorPaginationSummary,
  DataTableLoadingSlots,
  DataTableSortingMode,
} from "@/types/data-table.types"
import { ScanHistoryToolbar } from "./scan-history-data-table-sections"
import { useScanHistoryDataTableState } from "./scan-history-data-table-state"
import type { ScanStatusFilter } from "./scan-history-list-view-state"

interface ScanHistoryDataTableProps {
  data: ScanRecord[]
  columns: ColumnDef<ScanRecord>[]
  onAddNew?: () => void
  onBulkDelete?: () => void
  bulkDeleteLabel?: string
  selectedRowActions?: SelectedRowActionBarAction[]
  rowSelection?: Record<string, boolean>
  onRowSelectionChange?: (selection: Record<string, boolean>) => void
  selectedRows?: ScanRecord[]
  searchPlaceholder?: string
  searchValue?: string
  onSearch?: (value: string) => void
  isSearching?: boolean
  addButtonText?: string
  pagination?: { pageIndex: number; pageSize: number }
  setPagination?: React.Dispatch<React.SetStateAction<{ pageIndex: number; pageSize: number }>>
  paginationInfo?: PaginationInfo
  cursorPaginationSummary?: CursorPaginationSummary
  paginationNavigation?: CursorPaginationNavigation
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
  sortingMode?: DataTableSortingMode
  sorting?: SortingState
  onSortingChange?: (sorting: SortingState) => void
  hideToolbar?: boolean
  hidePagination?: boolean
  pageSizeOptions?: number[]
  statusFilter?: ScanStatusFilter
  onStatusFilterChange?: (status: ScanStatusFilter) => void
  onRowClick?: (scan: ScanRecord) => void
  rowActionLabel?: string
  loading?: boolean
  initialLoading?: boolean
  loadingRowCount?: number
  stableSurfaceRowCount?: number
  loadingSlots?: DataTableLoadingSlots
}

export function ScanHistoryDataTable({
  data = [],
  columns,
  onAddNew,
  onBulkDelete,
  bulkDeleteLabel,
  selectedRowActions,
  rowSelection,
  onRowSelectionChange,
  selectedRows,
  searchPlaceholder,
  searchValue,
  onSearch,
  isSearching = false,
  addButtonText,
  pagination: externalPagination,
  setPagination: setExternalPagination,
  paginationInfo,
  cursorPaginationSummary,
  paginationNavigation,
  onPaginationChange,
  sortingMode = "none",
  sorting,
  onSortingChange,
  hideToolbar = false,
  hidePagination = false,
  pageSizeOptions,
  statusFilter = [],
  onStatusFilterChange,
  onRowClick,
  rowActionLabel,
  loading = false,
  initialLoading = false,
  loadingRowCount,
  stableSurfaceRowCount,
  loadingSlots,
}: ScanHistoryDataTableProps) {
  const state = useScanHistoryDataTableState({ searchValue, onSearch })
  const toolbar = (
    <ScanHistoryToolbar
      state={state}
      loading={isSearching}
      placeholder={searchPlaceholder}
      status={statusFilter}
      onStatusChange={onStatusFilterChange}
    />
  )

  return (
    <BusinessListDataTable
      data={data}
      columns={columns}
      getRowId={(row) => String(row.id)}
      state={{
        pagination: externalPagination,
        setPagination: setExternalPagination,
        paginationInfo,
        cursorPaginationSummary,
        paginationNavigation,
        onPaginationChange,
        sortingMode,
        sorting,
        onSortingChange,
        rowSelection,
        onRowSelectionChange,
        selectedRows,
        isSearching,
      }}
      behavior={{
        enableAutoColumnSizing: true,
        onRowClick: onRowClick ? (row) => onRowClick(row as ScanRecord) : undefined,
        getRowActionLabel: onRowClick && rowActionLabel ? () => rowActionLabel : undefined,
      }}
      actions={{
        onBulkDelete,
        bulkDeleteLabel: bulkDeleteLabel ?? state.tActions("delete"),
        selectedRowActions,
        onAddNew,
        addButtonLabel: addButtonText || state.tScan("title"),
      }}
      ui={{
        hidePagination,
        pageSizeOptions,
        hideToolbar,
        emptyMessage: state.t("noData"),
        toolbarLeft: toolbar,
        loading,
        loadingPresentation: initialLoading ? "initial" : "rows",
        loadingRowCount,
        stableSurfaceRowCount,
        loadingSlots,
      }}
    />
  )
}
