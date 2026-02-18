"use client"

import type { ColumnDef } from "@tanstack/react-table"
import { UnifiedDataTable } from "@/components/ui/data-table/unified-data-table"
import type { FilterField } from "@/components/common/smart-filter-input"
import type { DownloadOption, PaginationState } from "@/types/data-table.types"
import type { PaginationInfo } from "@/types/common.types"

type SmartFilterDataTableProps<TData> = {
  data: TData[]
  columns: ColumnDef<TData>[]
  getRowId?: (row: TData) => string
  filterFields: FilterField[]
  filterExamples?: string[]
  filterValue?: string
  onFilterChange?: (value: string) => void
  isSearching?: boolean
  pagination?: PaginationState
  setPagination?: React.Dispatch<React.SetStateAction<PaginationState>>
  paginationInfo?: PaginationInfo
  onPaginationChange?: (pagination: PaginationState) => void
  onSelectionChange?: (selectedRows: TData[]) => void
  onBulkDelete?: () => void
  bulkDeleteLabel?: string
  onBulkAdd?: () => void
  bulkAddLabel?: string
  onAddNew?: () => void
  onAddHover?: () => void
  addButtonLabel?: string
  showAddButton?: boolean
  downloadOptions?: DownloadOption[]
  emptyMessage?: string
  className?: string
  tableClassName?: string
  hideToolbar?: boolean
  hidePagination?: boolean
}

export function SmartFilterDataTable<TData>({
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
  onPaginationChange,
  onSelectionChange,
  onBulkDelete,
  bulkDeleteLabel,
  onBulkAdd,
  bulkAddLabel,
  onAddNew,
  onAddHover,
  addButtonLabel,
  showAddButton,
  downloadOptions,
  emptyMessage,
  className,
  tableClassName,
  hideToolbar,
  hidePagination,
}: SmartFilterDataTableProps<TData>) {
  const handleSmartSearch = (rawQuery: string) => {
    onFilterChange?.(rawQuery)
  }

  const shouldShowAddButton = showAddButton ?? Boolean(onAddNew)

  return (
    <UnifiedDataTable
      data={data}
      columns={columns}
      getRowId={getRowId}
      state={{
        pagination,
        setPagination,
        paginationInfo,
        onPaginationChange,
        searchValue: filterValue,
        isSearching,
        onSelectionChange,
      }}
      behavior={{
        searchMode: "smart",
        onSearch: handleSmartSearch,
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
        downloadOptions: downloadOptions && downloadOptions.length > 0 ? downloadOptions : undefined,
      }}
      ui={{
        filterFields,
        filterExamples,
        emptyMessage,
        className,
        tableClassName,
        hideToolbar,
        hidePagination,
      }}
    />
  )
}
