"use client"

import { BusinessListDataTable } from "@/components/shared/data-table"
import { SimpleSearchToolbar } from "@/components/shared/data-table/simple-search-toolbar"
import { TargetTypeFacetedFilter } from "@/components/shared/data-table/target-type-filter-select"
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
  loading = false,
  loadingRowCount,
  stableSurfaceRowCount,
}: TargetsDataTableProps) {
  const state = useOrganizationTargetsDataTableState({ searchValue, onSearch })
  const hasQueryControls = Boolean(onSearch || onTypeFilterChange)

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
            after={onTypeFilterChange ? (
              <TargetTypeFacetedFilter
                title={state.tColumns("common.type")}
                values={typeFilter ?? []}
                onValuesChange={onTypeFilterChange}
                labels={{
                  all: state.tCommon("actions.all"),
                  domain: state.tTarget("types.domain"),
                  ip: state.tTarget("types.ip"),
                  cidr: state.tTarget("types.cidr"),
                }}
                clearLabel={state.tDataTable("clearFilter")}
              />
            ) : null}
          />
        ) : undefined,
      }}
    />
  )
}
