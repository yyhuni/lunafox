"use client"

import * as React from "react"
import type { ColumnDef, SortingState } from "@tanstack/react-table"
import { useTranslations } from "next-intl"
import { BusinessListDataTable } from "@/components/shared/data-table"
import { buildExportOptions } from "@/components/shared/data-table/data-table-export-helpers"
import { SimpleSearchToolbar } from "@/components/shared/data-table/simple-search-toolbar"
import { useSimpleSearchState } from "@/components/shared/data-table/use-simple-search"
import type { Subdomain } from "@/types/subdomain.types"
import type {
  CursorPaginationNavigation,
  CursorPaginationSummary,
  DataTableSortingMode,
} from "@/types/data-table.types"

// Component props type definition
interface SubdomainsDataTableProps {
  data: Subdomain[]
  columns: ColumnDef<Subdomain>[]
  onAddNew?: () => void
  onBulkAdd?: () => void
  onBulkDelete?: () => void
  onSelectionChange?: (selectedRows: Subdomain[]) => void
  selectedRows?: Subdomain[]
  searchValue?: string
  onSearch?: (value: string) => void
  isSearching?: boolean
  addButtonText?: string
  // Export callback functions
  onExportAll?: () => void
  onExportInteresting?: () => void
  onExportImportant?: () => void
  onExportSelected?: () => void
  // Server-side pagination support
  pagination?: { pageIndex: number; pageSize: number }
  setPagination?: React.Dispatch<React.SetStateAction<{ pageIndex: number; pageSize: number }>>
  cursorPaginationSummary?: CursorPaginationSummary
  paginationNavigation?: CursorPaginationNavigation
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
  sortingMode?: DataTableSortingMode
  sorting?: SortingState
  onSortingChange?: (sorting: SortingState) => void
  loading?: boolean
  initialLoading?: boolean
  loadingRowCount?: number
}

/**
 * Subdomain data table component
 * Uses UnifiedDataTable unified component
 */
export function SubdomainsDataTable({
  data = [],
  columns,
  onAddNew,
  onBulkAdd,
  onBulkDelete,
  onSelectionChange,
  selectedRows,
  searchValue,
  onSearch,
  isSearching = false,
  addButtonText = "Add",
  onExportAll,
  onExportInteresting,
  onExportImportant,
  onExportSelected,
  pagination: externalPagination,
  setPagination: setExternalPagination,
  cursorPaginationSummary,
  paginationNavigation,
  onPaginationChange,
  sortingMode = "none",
  sorting,
  onSortingChange,
  loading = false,
  initialLoading = false,
  loadingRowCount,
}: SubdomainsDataTableProps) {
  const t = useTranslations("common.status")
  const tActions = useTranslations("common.actions")
  const tExport = useTranslations("common.export")
  const {
    value: localSearchValue,
    handleSearchInputChange,
    commitNow: commitSearch,
  } = useSimpleSearchState({ searchValue, onSearch })

  // Export options
  const exportOptions = buildExportOptions(tExport, {
    onExportAll,
    onExportSelected,
    onExportImportant,
    onExportInteresting,
  })

  return (
    <BusinessListDataTable
      data={data}
      columns={columns}
      getRowId={(row) => String(row.id)}
      state={{
        pagination: externalPagination,
        setPagination: setExternalPagination,
        cursorPaginationSummary,
        paginationNavigation,
        onPaginationChange,
        sortingMode,
        sorting,
        onSortingChange,
        onSelectionChange,
        selectedRows,
      }}
      actions={{
        onBulkDelete,
        bulkDeleteLabel: tActions("delete"),
        onAddNew,
        addButtonLabel: addButtonText,
        showAddButton: !!onAddNew,
        onBulkAdd,
        bulkAddLabel: tActions("add"),
        showBulkAdd: !!onBulkAdd,
        exportOptions: exportOptions.length > 0 ? exportOptions : undefined,
      }}
      behavior={{
        expandColumnIds: ["name"],
      }}
      ui={{
        toolbarLeft: (
          <SimpleSearchToolbar
            value={localSearchValue}
            onChange={handleSearchInputChange}
            onSubmit={commitSearch}
            loading={isSearching}
            placeholder={tActions("searchSubdomain")}
            toolbarDensity="compact"
          />
        ),
        emptyMessage: t("noData"),
        showColumnVisibility: false,
        loading,
        loadingPresentation: initialLoading ? "initial" : "rows",
        loadingRowCount,
      }}
    />
  )
}
