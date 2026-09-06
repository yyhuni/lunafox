"use client"

import * as React from "react"
import type { ColumnDef, SortingState, VisibilityState } from "@tanstack/react-table"
import { useTranslations } from "next-intl"
import { BusinessListDataTable } from "@/components/shared/data-table"
import { buildExportOptions } from "@/components/shared/data-table/data-table-export-helpers"
import {
  DataTableFacetPanel,
  type DataTableFacetPanelFacet,
  type DataTableFacetedFilterOption,
} from "@/components/shared/data-table/faceted-filter"
import { SimpleSearchToolbar } from "@/components/shared/data-table/simple-search-toolbar"
import { useSimpleSearchState } from "@/components/shared/data-table/use-simple-search"
import type { WebSite } from "@/types/website.types"
import type {
  CursorPaginationNavigation,
  CursorPaginationSummary,
  DataTableSortingMode,
} from "@/types/data-table.types"
import { TABLE_DENSE_ROW_ESTIMATED_HEIGHT_PX } from "@/components/ui/table"

interface WebSitesDataTableProps {
  data: WebSite[]
  columns: ColumnDef<WebSite>[]
  searchValue?: string
  onSearchChange?: (value: string) => void
  isSearching?: boolean
  statusCodeFilter?: string[]
  onStatusCodeFilterChange?: (values: string[]) => void
  statusCodeOptions?: Array<DataTableFacetedFilterOption<string>>
  techFilter?: string[]
  onTechFilterChange?: (values: string[]) => void
  techOptions?: Array<DataTableFacetedFilterOption<string>>
  webserverFilter?: string[]
  onWebserverFilterChange?: (values: string[]) => void
  webserverOptions?: Array<DataTableFacetedFilterOption<string>>
  contentTypeFilter?: string[]
  onContentTypeFilterChange?: (values: string[]) => void
  contentTypeOptions?: Array<DataTableFacetedFilterOption<string>>
  vhostFilter?: string[]
  onVhostFilterChange?: (values: string[]) => void
  vhostOptions?: Array<DataTableFacetedFilterOption<string>>
  pagination?: { pageIndex: number; pageSize: number }
  setPagination?: React.Dispatch<React.SetStateAction<{ pageIndex: number; pageSize: number }>>
  cursorPaginationSummary?: CursorPaginationSummary
  paginationNavigation?: CursorPaginationNavigation
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
  sortingMode?: DataTableSortingMode
  sorting?: SortingState
  onSortingChange?: (sorting: SortingState) => void
  onBulkDelete?: () => void
  onSelectionChange?: (selectedRows: WebSite[]) => void
  selectedRows?: WebSite[]
  onExportAll?: () => void
  onExportSelected?: () => void
  onBulkAdd?: () => void
  onRowClick?: (website: WebSite) => void
  columnVisibility?: VisibilityState
  onColumnVisibilityChange?: (visibility: VisibilityState) => void
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

export function WebSitesDataTable({
  data = [],
  columns,
  searchValue = "",
  onSearchChange,
  isSearching = false,
  statusCodeFilter = [],
  onStatusCodeFilterChange,
  statusCodeOptions = [],
  techFilter = [],
  onTechFilterChange,
  techOptions = [],
  webserverFilter = [],
  onWebserverFilterChange,
  webserverOptions = [],
  contentTypeFilter = [],
  onContentTypeFilterChange,
  contentTypeOptions = [],
  vhostFilter = [],
  onVhostFilterChange,
  vhostOptions = [
    { value: "true", label: "true" },
    { value: "false", label: "false" },
  ],
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
  onRowClick,
  columnVisibility,
  onColumnVisibilityChange,
  loading = false,
  initialLoading = false,
  loadingRowCount,
}: WebSitesDataTableProps) {
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

  const mergedStatusCodeOptions = React.useMemo(
    () => mergeSelectedFilterOptions(statusCodeOptions, statusCodeFilter),
    [statusCodeFilter, statusCodeOptions]
  )
  const mergedTechOptions = React.useMemo(
    () => mergeSelectedFilterOptions(techOptions, techFilter),
    [techFilter, techOptions]
  )
  const mergedWebserverOptions = React.useMemo(
    () => mergeSelectedFilterOptions(webserverOptions, webserverFilter),
    [webserverFilter, webserverOptions]
  )
  const mergedContentTypeOptions = React.useMemo(
    () => mergeSelectedFilterOptions(contentTypeOptions, contentTypeFilter),
    [contentTypeFilter, contentTypeOptions]
  )
  const mergedVhostOptions = React.useMemo(
    () => mergeSelectedFilterOptions(vhostOptions, vhostFilter),
    [vhostFilter, vhostOptions]
  )
  const activeFilterCount = statusCodeFilter.length
    + techFilter.length
    + webserverFilter.length
    + contentTypeFilter.length
    + vhostFilter.length

  const websiteFacetPanelItems: DataTableFacetPanelFacet[] = [
    {
      id: "statusCode",
      label: tColumns("website.statusCode"),
      values: statusCodeFilter,
      onValuesChange: onStatusCodeFilterChange ?? (() => undefined),
      options: mergedStatusCodeOptions,
      emptyLabel: t("noData"),
      clearLabel: tDataTable("clearFilter"),
    },
    {
      id: "tech",
      label: tColumns("endpoint.technologies"),
      values: techFilter,
      onValuesChange: onTechFilterChange ?? (() => undefined),
      options: mergedTechOptions,
      emptyLabel: t("noData"),
      clearLabel: tDataTable("clearFilter"),
    },
    {
      id: "webserver",
      label: tColumns("endpoint.webServer"),
      values: webserverFilter,
      onValuesChange: onWebserverFilterChange ?? (() => undefined),
      options: mergedWebserverOptions,
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
    {
      id: "vhost",
      label: tColumns("endpoint.vhost"),
      values: vhostFilter,
      onValuesChange: onVhostFilterChange ?? (() => undefined),
      options: mergedVhostOptions,
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
        facets={websiteFacetPanelItems}
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
        columnVisibility,
        onColumnVisibilityChange,
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
        onRowClick: onRowClick
          ? (row) => onRowClick(row as WebSite)
          : undefined,
        getRowActionLabel: (row) => `${tActions("details")}: ${(row as WebSite).url}`,
      }}
      ui={{
        toolbarLeft,
        emptyMessage: t("noData"),
        showColumnVisibility: true,
        loading,
        loadingPresentation: initialLoading ? "initial" : "rows",
        initialLoadingToolbarFilterCount: 1,
        loadingRowCount,
        loadingRowHeightEstimate: TABLE_DENSE_ROW_ESTIMATED_HEIGHT_PX,
      }}
    />
  )
}
