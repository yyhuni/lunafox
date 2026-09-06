"use client"

import * as React from "react"
import type { ColumnDef, SortingState } from "@tanstack/react-table"
import { useTranslations } from "next-intl"
import { getTranslatedFields } from "@/components/common/smart-filter-input"
import { BusinessListDataTable } from "@/components/shared/data-table"
import { buildExportOptions } from "@/components/shared/data-table/data-table-export-helpers"
import { DataTableFacetedFilter, DataTableFacetedFilterGroup, type DataTableFacetedFilterOption } from "@/components/shared/data-table/faceted-filter"
import { SimpleSearchToolbar } from "@/components/shared/data-table/simple-search-toolbar"
import { useSimpleSearchState } from "@/components/shared/data-table/use-simple-search"
import type { IPAddress } from "@/types/ip-address.types"
import type {
  CursorPaginationNavigation,
  CursorPaginationSummary,
  DataTableSortingMode,
} from "@/types/data-table.types"

interface IPAddressesDataTableProps {
  data: IPAddress[]
  columns: ColumnDef<IPAddress>[]
  filterValue?: string
  onFilterChange?: (value: string) => void
  searchPlaceholder?: string
  portFilter?: string[]
  onPortFilterChange?: (values: string[]) => void
  portOptions?: Array<DataTableFacetedFilterOption<string>>
  pagination?: { pageIndex: number; pageSize: number }
  setPagination?: React.Dispatch<React.SetStateAction<{ pageIndex: number; pageSize: number }>>
  cursorPaginationSummary?: CursorPaginationSummary
  paginationNavigation?: CursorPaginationNavigation
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
  sortingMode?: DataTableSortingMode
  sorting?: SortingState
  onSortingChange?: (sorting: SortingState) => void
  onBulkDelete?: () => void
  onSelectionChange?: (selectedRows: IPAddress[]) => void
  selectedRows?: IPAddress[]
  onExportAll?: () => void
  onExportSelected?: () => void
  loading?: boolean
  initialLoading?: boolean
  loadingRowCount?: number
}

const noopPortFilterChange = () => {}

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
  return Array.from(optionByValue.values()).sort((left, right) => Number(left.value) - Number(right.value))
}

export function IPAddressesDataTable({
  data = [],
  columns,
  filterValue = "",
  onFilterChange,
  searchPlaceholder,
  portFilter = [],
  onPortFilterChange = noopPortFilterChange,
  portOptions = [],
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
  loading = false,
  initialLoading = false,
  loadingRowCount,
}: IPAddressesDataTableProps) {
  const t = useTranslations("common.status")
  const tExport = useTranslations("common.export")
  const tActions = useTranslations("common.actions")
  const tDataTable = useTranslations("dataTable")
  const tFilter = useTranslations("filter")
  const translatedFields = React.useMemo(() => getTranslatedFields(tFilter), [tFilter])
  const {
    value: localSearchValue,
    handleSearchInputChange,
    commitNow: commitSearch,
  } = useSimpleSearchState({ searchValue: filterValue, onSearch: onFilterChange })

  const mergedPortOptions = React.useMemo(
    () => mergeSelectedFilterOptions(portOptions, portFilter),
    [portFilter, portOptions]
  )
  const toolbarLeft = (
    <div className="flex w-full min-w-0 flex-wrap items-center gap-2">
      <SimpleSearchToolbar
        value={localSearchValue}
        onChange={handleSearchInputChange}
        onSubmit={commitSearch}
        placeholder={searchPlaceholder ?? tActions("searchIPOrHost")}
        toolbarDensity="compact"
      />
      <DataTableFacetedFilterGroup hasSelectedValues={portFilter.length > 0} onReset={() => onPortFilterChange?.([])}>
        <DataTableFacetedFilter
          title={translatedFields.port.label}
          values={portFilter}
          onValuesChange={onPortFilterChange}
          options={mergedPortOptions}
          emptyLabel={t("noData")}
          clearLabel={tDataTable("clearFilter")}
        />
      </DataTableFacetedFilterGroup>
    </div>
  )

  // Export options
  const exportOptions = buildExportOptions(tExport, {
    onExportAll,
    onExportSelected,
  })

  return (
    <BusinessListDataTable
      data={data}
      columns={columns}
      getRowId={(row) => row.ip}
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
        showAddButton: false,
        exportOptions: exportOptions.length > 0 ? exportOptions : undefined,
      }}
      behavior={{
        expandColumnIds: ["hosts", "ports"],
      }}
      ui={{
        toolbarLeft,
        emptyMessage: t("noData"),
        showColumnVisibility: false,
        loading,
        loadingPresentation: initialLoading ? "initial" : "rows",
        initialLoadingToolbarFilterCount: 1,
        loadingRowCount,
      }}
    />
  )
}
