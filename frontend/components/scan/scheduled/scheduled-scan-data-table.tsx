"use client"

import type { ColumnDef } from "@tanstack/react-table"

import { UnifiedDataTable } from "@/components/ui/data-table/unified-data-table"
import { SimpleSearchToolbar } from "@/components/ui/data-table/simple-search-toolbar"
import { useScheduledScanDataTableState } from "./scheduled-scan-data-table-state"

import type { ScheduledScan } from "@/types/scheduled-scan.types"

interface ScheduledScanDataTableProps {
  data: ScheduledScan[]
  columns: ColumnDef<ScheduledScan>[]
  onAddNew?: () => void
  searchPlaceholder?: string
  searchValue?: string
  onSearch?: (value: string) => void
  isSearching?: boolean
  addButtonText?: string
  page?: number
  pageSize?: number
  total?: number
  totalPages?: number
  onPageChange?: (page: number) => void
  onPageSizeChange?: (pageSize: number) => void
}

/**
 * Scheduled scan data table component
 * Uses UnifiedDataTable unified component
 */
export function ScheduledScanDataTable({
  data = [],
  columns,
  onAddNew,
  searchPlaceholder,
  searchValue,
  onSearch,
  isSearching = false,
  addButtonText,
  page = 1,
  pageSize = 10,
  total = 0,
  totalPages = 1,
  onPageChange,
  onPageSizeChange,
}: ScheduledScanDataTableProps) {
  const state = useScheduledScanDataTableState({
    searchValue,
    onSearch,
    page,
    pageSize,
    total,
    totalPages,
  })

  const handlePaginationChange = (newPagination: { pageIndex: number; pageSize: number }) => {
    if (newPagination.pageSize !== pageSize && onPageSizeChange) {
      onPageSizeChange(newPagination.pageSize)
    }
    if (newPagination.pageIndex !== page - 1 && onPageChange) {
      onPageChange(newPagination.pageIndex + 1)
    }
  }

  return (
    <UnifiedDataTable
      data={data}
      columns={columns}
      getRowId={(row) => String(row.id)}
      state={{
        pagination: state.pagination,
        paginationInfo: state.paginationInfo,
        onPaginationChange: handlePaginationChange,
      }}
      behavior={{ enableRowSelection: false }}
      actions={{
        showBulkDelete: false,
        onAddNew,
        addButtonLabel: addButtonText || state.tScan("createTitle"),
      }}
      ui={{
        emptyMessage: state.t("noData"),
        toolbarLeft: (
          <SimpleSearchToolbar
            value={state.localSearchValue}
            onChange={state.setLocalSearchValue}
            onSubmit={state.handleSearchSubmit}
            loading={isSearching}
            placeholder={searchPlaceholder || state.tScan("searchPlaceholder")}
          />
        ),
      }}
    />
  )
}
