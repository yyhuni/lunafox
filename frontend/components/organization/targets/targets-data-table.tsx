"use client"

import { BusinessListDataTable } from "@/components/shared/data-table"
import { DataTableFacetPanel, type DataTableFacetPanelFacet } from "@/components/shared/data-table/faceted-filter"
import { SimpleSearchToolbar } from "@/components/shared/data-table/simple-search-toolbar"
import type { OrganizationTargetScannedFilter } from "@/components/organization/organization-detail-view-state"
import type { ColumnDef } from "@tanstack/react-table"
import type { Target } from "@/types/target.types"
import type { TargetType } from "@/types/target.types"
import type {
  CursorPaginationNavigation,
  CursorPaginationSummary,
} from "@/types/data-table.types"
import { useOrganizationTargetsDataTableState } from "./targets-data-table-state"

interface TargetsDataTableProps {
  data: Target[]
  columns: ColumnDef<Target>[]
  onAddNew?: () => void
  onAddHover?: () => void
  onBulkDelete?: () => void
  onSelectionChange?: (selectedRows: Target[]) => void
  selectedRows?: Target[]
  searchPlaceholder?: string
  searchValue?: string
  onSearch?: (value: string) => void
  isSearching?: boolean
  addButtonText?: string
  pagination?: { pageIndex: number; pageSize: number }
  cursorPaginationSummary?: CursorPaginationSummary
  paginationNavigation?: CursorPaginationNavigation
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
  typeFilter?: TargetType[]
  onTypeFilterChange?: (value: TargetType[]) => void
  scannedFilter?: OrganizationTargetScannedFilter
  onScannedFilterChange?: (value: OrganizationTargetScannedFilter) => void
  loading?: boolean
  loadingRowCount?: number
  stableSurfaceRowCount?: number
}

/**
 * Target data table component (organization version)
 */
export function TargetsDataTable({
  data = [],
  columns,
  onAddNew,
  onAddHover,
  onBulkDelete,
  onSelectionChange,
  selectedRows,
  searchPlaceholder,
  searchValue,
  onSearch,
  isSearching = false,
  addButtonText,
  pagination: externalPagination,
  cursorPaginationSummary,
  paginationNavigation,
  onPaginationChange,
  typeFilter,
  onTypeFilterChange,
  scannedFilter,
  onScannedFilterChange,
  loading = false,
  loadingRowCount,
  stableSurfaceRowCount,
}: TargetsDataTableProps) {
  const state = useOrganizationTargetsDataTableState({ searchValue, onSearch })
  const hasQueryControls = Boolean(onSearch || onTypeFilterChange || onScannedFilterChange)
  const targetFilterFacets: DataTableFacetPanelFacet[] = []
  if (onTypeFilterChange) {
    targetFilterFacets.push({
      id: "type",
      label: state.tColumns("common.type"),
      values: typeFilter ?? [],
      onValuesChange: (values) => onTypeFilterChange(values.filter((value): value is TargetType => (
        value === "domain" || value === "ip" || value === "cidr"
      ))),
      options: [
        { value: "domain", label: state.tTarget("types.domain") },
        { value: "ip", label: state.tTarget("types.ip") },
        { value: "cidr", label: state.tTarget("types.cidr") },
      ],
      emptyLabel: state.tCommon("actions.all"),
      clearLabel: state.tDataTable("clearFilter"),
    })
  }
  if (onScannedFilterChange) {
    targetFilterFacets.push({
      id: "scanned",
      label: state.tTarget("scannedFilter.title"),
      values: scannedFilter ?? [],
      onValuesChange: (values) => onScannedFilterChange(values.filter((value): value is OrganizationTargetScannedFilter[number] => (
        value === "true" || value === "false"
      ))),
      options: [
        { value: "true", label: state.tTarget("scannedFilter.scanned") },
        { value: "false", label: state.tTarget("scannedFilter.notScanned") },
      ],
      emptyLabel: state.tCommon("actions.all"),
      clearLabel: state.tDataTable("clearFilter"),
    })
  }

  return (
    <BusinessListDataTable
      data={data}
      columns={columns}
      getRowId={(row) => String(row.id)}
      state={{
        pagination: externalPagination,
        cursorPaginationSummary,
        paginationNavigation,
        onPaginationChange,
        onSelectionChange,
        selectedRows,
      }}
      behavior={{
        expandColumnIds: ["name"],
      }}
      actions={{
        showBulkDelete: !!onBulkDelete,
        onBulkDelete,
        bulkDeleteLabel: state.tTooltips("unlinkTarget"),
        showAddButton: !!onAddNew,
        onAddNew,
        onAddHover,
        addButtonLabel: addButtonText || state.tTarget("addTarget"),
      }}
      ui={{
        emptyMessage: state.t("noData"),
        showColumnVisibility: false,
        loading,
        loadingRowCount,
        stableSurfaceRowCount,
        toolbarLeft: hasQueryControls ? (
          <SimpleSearchToolbar
            value={state.localSearchValue}
            onChange={state.handleSearchInputChange}
            onSubmit={state.commitSearch}
            loading={isSearching}
            placeholder={searchPlaceholder || state.tTarget("title")}
            after={targetFilterFacets.length > 0 ? (
              <DataTableFacetPanel
                title={state.tDataTable("filter")}
                activeCount={(typeFilter?.length ?? 0) + (scannedFilter?.length ?? 0)}
                facets={targetFilterFacets}
              />
            ) : null}
          />
        ) : undefined,
      }}
    />
  )
}
