"use client"

import type { ColumnDef } from "@tanstack/react-table"
import type { Dispatch, SetStateAction } from "react"

import { SmartFilterBusinessListDataTable } from "./smart-filter-business-list-data-table"
import type { FilterField } from "@/components/common/smart-filter-input"
import type {
  CursorPaginationNavigation,
  CursorPaginationSummary,
  ExportOption,
  PaginationState,
} from "@/types/data-table.types"
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
  expandColumnIds?: string[]
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
  expandColumnIds,
}: SmartFilterDataTableProps<TData>) {
  const handleSmartSearch = (rawQuery: string) => {
    onFilterChange?.(rawQuery)
  }

  return (
    <SmartFilterBusinessListDataTable
      data={data}
      columns={columns}
      getRowId={getRowId ? (row) => getRowId(row) : undefined}
      filterFields={filterFields}
      filterExamples={filterExamples}
      filterValue={filterValue}
      onFilterChange={handleSmartSearch}
      isSearching={isSearching}
      pagination={pagination}
      setPagination={setPagination}
      paginationInfo={paginationInfo}
      cursorPaginationSummary={cursorPaginationSummary}
      paginationNavigation={paginationNavigation}
      onPaginationChange={onPaginationChange}
      onSelectionChange={onSelectionChange}
      selectedRows={selectedRows}
      onBulkDelete={onBulkDelete}
      bulkDeleteLabel={bulkDeleteLabel}
      onBulkAdd={onBulkAdd}
      bulkAddLabel={bulkAddLabel}
      onAddNew={onAddNew}
      onAddHover={onAddHover}
      addButtonLabel={addButtonLabel}
      showAddButton={showAddButton}
      exportOptions={exportOptions}
      emptyMessage={emptyMessage}
      className={className}
      tableClassName={tableClassName}
      hideToolbar={hideToolbar}
      hidePagination={hidePagination}
      expandColumnIds={expandColumnIds}
    />
  )
}
