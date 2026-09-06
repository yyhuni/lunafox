"use client"

import * as React from "react"
import type { ColumnDef, SortingState } from "@tanstack/react-table"
import { useTranslations } from "next-intl"
import { BusinessListDataTable } from "@/components/shared/data-table"
import { buildExportOptions } from "@/components/shared/data-table/data-table-export-helpers"
import { DataTableFacetPanel, type DataTableFacetPanelFacet, type DataTableFacetedFilterOption } from "@/components/shared/data-table/faceted-filter"
import { SimpleSearchToolbar } from "@/components/shared/data-table/simple-search-toolbar"
import { useSimpleSearchState } from "@/components/shared/data-table/use-simple-search"
import type { Directory } from "@/types/directory.types"
import type {
  CursorPaginationNavigation,
  CursorPaginationSummary,
  DataTableSortingMode,
} from "@/types/data-table.types"

interface DirectoriesDataTableProps {
  data: Directory[]
  columns: ColumnDef<Directory>[]
  searchValue?: string
  onSearchChange?: (value: string) => void
  isSearching?: boolean
  statusFilter?: string[]
  onStatusFilterChange?: (values: string[]) => void
  statusOptions?: Array<DataTableFacetedFilterOption<string>>
  contentTypeFilter?: string[]
  onContentTypeFilterChange?: (values: string[]) => void
  contentTypeOptions?: Array<DataTableFacetedFilterOption<string>>
  pagination?: { pageIndex: number; pageSize: number }
  setPagination?: React.Dispatch<React.SetStateAction<{ pageIndex: number; pageSize: number }>>
  cursorPaginationSummary?: CursorPaginationSummary
  paginationNavigation?: CursorPaginationNavigation
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
  sortingMode?: DataTableSortingMode
  sorting?: SortingState
  onSortingChange?: (sorting: SortingState) => void
  onBulkDelete?: () => void
  onSelectionChange?: (selectedRows: Directory[]) => void
  selectedRows?: Directory[]
  onExportAll?: () => void
  onExportSelected?: () => void
  onBulkAdd?: () => void
  loading?: boolean
  initialLoading?: boolean
  loadingRowCount?: number
}

function mergeSelectedFilterOptions(
  options: Array<DataTableFacetedFilterOption<string>>,
  selected: string[]
) {
  const optionByValue = new Map(options.map((option) => [option.value, option]))
  for (const value of selected) {
    const trimmed = value.trim()
    if (trimmed && !optionByValue.has(trimmed)) {
      optionByValue.set(trimmed, { value: trimmed, label: trimmed })
    }
  }
  return Array.from(optionByValue.values()).sort((left, right) => left.label.localeCompare(right.label, undefined, { numeric: true }))
}

export function DirectoriesDataTable({
  data = [],
  columns,
  searchValue = "",
  onSearchChange,
  isSearching = false,
  statusFilter = [],
  onStatusFilterChange,
  statusOptions = [],
  contentTypeFilter = [],
  onContentTypeFilterChange,
  contentTypeOptions = [],
  pagination,
  setPagination,
  cursorPaginationSummary,
  paginationNavigation,
  onPaginationChange,
  sortingMode = "none",
  sorting,
  onSortingChange,
  onBulkDelete,
  onSelectionChange,
  selectedRows,
  onExportAll,
  onExportSelected,
  onBulkAdd,
  loading = false,
  initialLoading = false,
  loadingRowCount,
}: DirectoriesDataTableProps) {
  const t = useTranslations("common.status")
  const tActions = useTranslations("common.actions")
  const tExport = useTranslations("common.export")
  const tDataTable = useTranslations("dataTable")
  const tColumns = useTranslations("columns")
  const {
    value: localSearchValue,
    handleSearchInputChange,
    commitNow: commitSearch,
  } = useSimpleSearchState({ searchValue, onSearch: onSearchChange })

  const mergedStatusOptions = React.useMemo(
    () => mergeSelectedFilterOptions(statusOptions, statusFilter),
    [statusFilter, statusOptions]
  )
  const mergedContentTypeOptions = React.useMemo(
    () => mergeSelectedFilterOptions(contentTypeOptions, contentTypeFilter),
    [contentTypeFilter, contentTypeOptions]
  )
  const activeFilterCount = statusFilter.length + contentTypeFilter.length
  const directoryFacetPanelItems: DataTableFacetPanelFacet[] = [
    {
      id: "status",
      label: tColumns("common.status"),
      values: statusFilter,
      onValuesChange: onStatusFilterChange ?? (() => undefined),
      options: mergedStatusOptions,
      emptyLabel: t("noData"),
      clearLabel: tDataTable("clearFilter"),
    },
    {
      id: "contentType",
      label: tColumns("endpoint.contentType"),
      values: contentTypeFilter,
      onValuesChange: onContentTypeFilterChange ?? (() => undefined),
      options: mergedContentTypeOptions,
      emptyLabel: t("noData"),
      clearLabel: tDataTable("clearFilter"),
    },
  ]

  const toolbarLeft = (
    <div className="flex w-full min-w-0 flex-wrap items-center gap-2">
      <SimpleSearchToolbar
        value={localSearchValue}
        onChange={handleSearchInputChange}
        onSubmit={commitSearch}
        loading={isSearching}
        placeholder={tActions("searchURL")}
        toolbarDensity="compact"
      />
      <DataTableFacetPanel
        title={tDataTable("filter")}
        activeCount={activeFilterCount}
        facets={directoryFacetPanelItems}
      />
    </div>
  )

  const exportOptions = buildExportOptions(tExport, {
    onExportAll,
    onExportSelected,
  })

  return (
    <BusinessListDataTable
      data={data}
      columns={columns}
      getRowId={(row) => String(row.id)}
      state={{
        pagination,
        setPagination,
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
        onBulkAdd,
        bulkAddLabel: tActions("add"),
        showAddButton: false,
        exportOptions: exportOptions.length > 0 ? exportOptions : undefined,
      }}
      behavior={{
        expandColumnIds: ["url"],
      }}
      ui={{
        toolbarLeft,
        emptyMessage: t("noData"),
        showColumnVisibility: false,
        loading,
        loadingPresentation: initialLoading ? "initial" : "rows",
        loadingRowCount,
      }}
    />
  )
}
