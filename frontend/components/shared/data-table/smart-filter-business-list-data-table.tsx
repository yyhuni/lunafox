"use client"

import type { ColumnDef } from "@tanstack/react-table"
import type { Dispatch, ReactNode, SetStateAction } from "react"

import { BusinessListDataTable } from "./business-list-data-table"

import type { FilterField } from "@/components/common/smart-filter-input"
import type {
  CursorPaginationNavigation,
  CursorPaginationSummary,
  ExportOption,
  PaginationState,
  UnifiedDataTableStateConfig,
} from "@/types/data-table.types"
import type { PaginationInfo } from "@/types/common.types"

type SmartFilterBusinessListDataTableProps<TData> = {
  data: TData[]
  columns: ColumnDef<TData, unknown>[]
  getRowId?: (row: TData) => string
  filterFields: FilterField[]
  filterExamples?: string[]
  filterValue?: string
  onFilterChange?: (value: string) => void
  isSearching?: boolean
  pagination?: PaginationState
  setPagination?: Dispatch<SetStateAction<PaginationState>>
  paginationInfo?: PaginationInfo
  cursorPaginationSummary?: CursorPaginationSummary
  paginationNavigation?: CursorPaginationNavigation
  onPaginationChange?: (pagination: PaginationState) => void
  onSelectionChange?: (selectedRows: TData[]) => void
  selectedRows?: TData[]
  onBulkDelete?: () => void
  bulkDeleteLabel?: string
  onBulkAdd?: () => void
  bulkAddLabel?: string
  onAddNew?: () => void
  onAddHover?: () => void
  addButtonLabel?: string
  showAddButton?: boolean
  exportOptions?: ExportOption[]
  emptyMessage?: string
  className?: string
  tableClassName?: string
  hideToolbar?: boolean
  hidePagination?: boolean
  pageSizeOptions?: number[]
  toolbarLeft?: ReactNode
  toolbarRight?: ReactNode
  expandColumnIds?: string[]
}

export function SmartFilterBusinessListDataTable<TData>({
  data,
  columns,
  getRowId,
  filterFields,
  filterExamples,
  filterValue,
  onFilterChange,
  isSearching,
  pagination,
  setPagination,
  paginationInfo,
  cursorPaginationSummary,
  paginationNavigation,
  onPaginationChange,
  onSelectionChange,
  selectedRows,
  onBulkDelete,
  bulkDeleteLabel,
  onBulkAdd,
  bulkAddLabel,
  onAddNew,
  onAddHover,
  addButtonLabel,
  showAddButton,
  exportOptions,
  emptyMessage,
  className,
  tableClassName,
  hideToolbar,
  hidePagination,
  pageSizeOptions,
  toolbarLeft,
  toolbarRight,
  expandColumnIds,
}: SmartFilterBusinessListDataTableProps<TData>) {
  const shouldShowAddButton = showAddButton ?? Boolean(onAddNew)

  const state: UnifiedDataTableStateConfig<TData> = {
    pagination,
    setPagination,
    paginationInfo,
    cursorPaginationSummary,
    paginationNavigation,
    onPaginationChange,
    searchValue: filterValue,
    isSearching,
    onSelectionChange,
    selectedRows,
  }

  return (
    <BusinessListDataTable
      data={data}
      columns={columns}
      getRowId={getRowId}
      state={state}
      behavior={{
        searchMode: "smart",
        onSearch: onFilterChange,
        expandColumnIds,
      }}
      actions={{
        onBulkDelete,
        bulkDeleteLabel,
        onAddNew,
        onAddHover,
        addButtonLabel,
        showAddButton: shouldShowAddButton,
        onBulkAdd,
        bulkAddLabel,
        exportOptions: exportOptions && exportOptions.length > 0 ? exportOptions : undefined,
      }}
      ui={{
        filterFields,
        filterExamples,
        emptyMessage,
        className,
        tableClassName,
        hideToolbar,
        hidePagination,
        pageSizeOptions,
        toolbarLeft,
        toolbarRight,
      }}
    />
  )
}
