"use client"

import * as React from "react"
import type { ColumnDef } from "@tanstack/react-table"

import { UnifiedDataTable } from "@/components/ui/data-table/unified-data-table"

import type { ScanRecord, ScanStatus } from "@/types/scan.types"
import type { PaginationInfo } from "@/types/common.types"
import { ScanHistoryToolbar } from "./scan-history-data-table-sections"
import { useScanHistoryDataTableState } from "./scan-history-data-table-state"

interface ScanHistoryDataTableProps {
  data: ScanRecord[]
  columns: ColumnDef<ScanRecord>[]
  onAddNew?: () => void
  onBulkDelete?: () => void
  onSelectionChange?: (selectedRows: ScanRecord[]) => void
  searchPlaceholder?: string
  searchValue?: string
  onSearch?: (value: string) => void
  isSearching?: boolean
  addButtonText?: string
  pagination?: { pageIndex: number; pageSize: number }
  setPagination?: React.Dispatch<React.SetStateAction<{ pageIndex: number; pageSize: number }>>
  paginationInfo?: PaginationInfo
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
  hideToolbar?: boolean
  hidePagination?: boolean
  pageSizeOptions?: number[]
  statusFilter?: ScanStatus | "all"
  onStatusFilterChange?: (status: ScanStatus | "all") => void
}

/**
 * Scan history data table component
 * Uses UnifiedDataTable unified component
 */
export function ScanHistoryDataTable({
  data = [],
  columns,
  onAddNew,
  onBulkDelete,
  onSelectionChange,
  searchPlaceholder,
  searchValue,
  onSearch,
  isSearching = false,
  addButtonText,
  pagination: externalPagination,
  setPagination: setExternalPagination,
  paginationInfo,
  onPaginationChange,
  hideToolbar = false,
  hidePagination = false,
  pageSizeOptions,
  statusFilter = "all",
  onStatusFilterChange,
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
    <UnifiedDataTable
      data={data}
      columns={columns}
      getRowId={(row) => String(row.id)}
      state={{
        pagination: externalPagination,
        setPagination: setExternalPagination,
        paginationInfo,
        onPaginationChange,
        onSelectionChange,
        isSearching,
      }}
      behavior={{
        enableAutoColumnSizing: true,
        expandColumnIds: ["target", "cachedStats", "workflowNames"],
      }}
      actions={{
        onBulkDelete,
        bulkDeleteLabel: state.tActions("delete"),
        onAddNew,
        addButtonLabel: addButtonText || state.tScan("title"),
      }}
      ui={{
        hidePagination,
        pageSizeOptions,
        hideToolbar,
        emptyMessage: state.t("noData"),
        toolbarLeft: toolbar,
      }}
    />
  )
}
