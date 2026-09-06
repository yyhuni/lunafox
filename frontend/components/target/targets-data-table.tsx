"use client"

import { semanticIcons } from "@/components/icons"
import {
  SelectedRowActionBar,
  type SelectedRowActionBarAction,
} from "@/components/shared/data-table"
import { UnifiedDataTable } from "@/components/shared/data-table/unified-data-table"
import { SimpleSearchToolbar } from "@/components/shared/data-table/simple-search-toolbar"
import { TargetTypeFacetedFilter } from "@/components/shared/data-table/target-type-filter-select"
import type { ColumnDef, SortingState } from "@tanstack/react-table"
import type { Target } from "@/types/target.types"
import type { TargetType } from "@/types/target.types"
import type {
  CursorPaginationNavigation,
  DataTableLoadingSlots,
} from "@/types/data-table.types"
import { useTargetTargetsDataTableState } from "./targets-data-table-state"

const RunIcon = semanticIcons.action.run
const DeleteIcon = semanticIcons.action.delete

interface TargetsDataTableProps {
  data: Target[]
  columns: ColumnDef<Target>[]
  onAddNew?: () => void
  onAddHover?: () => void
  onBulkDelete?: () => void
  onBulkInitiateScan?: () => void
  onSelectionChange?: (selectedRows: Target[]) => void
  selectedRows?: Target[]
  searchPlaceholder?: string
  searchValue?: string
  onSearch?: (value: string) => void
  isSearching?: boolean
  addButtonText?: string
  // Pagination related props
  pagination?: { pageIndex: number, pageSize: number }
  paginationNavigation?: CursorPaginationNavigation
  onPaginationChange?: (pagination: { pageIndex: number, pageSize: number }) => void
  totalCount?: number
  manualPagination?: boolean
  sorting?: SortingState
  onSortingChange?: (sorting: SortingState) => void
  // Type filter
  typeFilter?: TargetType[]
  onTypeFilterChange?: (value: TargetType[]) => void
  // Styling
  className?: string
  tableClassName?: string
  hideToolbar?: boolean
  hidePagination?: boolean
  loading?: boolean
  loadingRowCount?: number
  stableSurfaceRowCount?: number
  loadingSlots?: DataTableLoadingSlots
}

/**
 * Targets data table component (target version)
 * Uses UnifiedDataTable unified component
 */
export function TargetsDataTable({
  data = [],
  columns,
  onAddNew,
  onAddHover,
  onBulkDelete,
  onBulkInitiateScan,
  onSelectionChange,
  selectedRows = [],
  searchPlaceholder,
  searchValue,
  onSearch,
  isSearching = false,
  addButtonText,
  pagination: externalPagination,
  paginationNavigation,
  onPaginationChange,
  totalCount,
  manualPagination = false,
  sorting,
  onSortingChange,
  typeFilter,
  onTypeFilterChange,
  className,
  tableClassName,
  hideToolbar = false,
  hidePagination = false,
  loading = false,
  loadingRowCount,
  stableSurfaceRowCount,
  loadingSlots,
}: TargetsDataTableProps) {
  const state = useTargetTargetsDataTableState({
    searchValue,
    onSearch,
    externalPagination,
    onPaginationChange,
    manualPagination,
    totalCount,
    sorting,
    onSortingChange,
  })
  const selectedRowActions: SelectedRowActionBarAction[] = [
    ...(onBulkInitiateScan
      ? [{
          key: "initiate-scan",
          label: state.tTooltips("initiateScan"),
          icon: RunIcon,
          group: "scan",
          onClick: onBulkInitiateScan,
        }]
      : []),
    ...(onBulkDelete
      ? [{
          key: "delete",
          label: state.tActions("delete"),
          icon: DeleteIcon,
          tone: "destructive" as const,
          group: "danger",
          onClick: onBulkDelete,
        }]
      : []),
  ]

  return (
    <>
      <UnifiedDataTable
        data={data}
        columns={columns}
        getRowId={(row) => String(row.id)}
        state={{
          pagination: state.pagination,
          setPagination: onPaginationChange ? undefined : state.setInternalPagination,
          cursorPaginationSummary: state.cursorPaginationSummary,
          paginationNavigation,
          onPaginationChange: state.handlePaginationChange,
          sortingMode: "server",
          sorting: state.sorting,
          onSortingChange: state.onSortingChange,
          onSelectionChange,
          selectedRows,
        }}
        behavior={{
          expandColumnIds: ["name", "organizations"],
          columnLayout: "fixed",
        }}
        actions={{
          showBulkDelete: false,
          onAddNew,
          onAddHover,
          addButtonLabel: addButtonText || state.tTarget("addTarget"),
          showAddButton: !!onAddNew,
        }}
        ui={{
          emptyMessage: state.t("noData"),
          showColumnVisibility: false,
          rowDensity: "comfortable",
          toolbarLeft: (
            <SimpleSearchToolbar
              value={state.localSearchValue}
              onChange={state.handleSearchInputChange}
              onSubmit={state.commitSearch}
              loading={isSearching}
              placeholder={searchPlaceholder || state.tTarget("title")}
              toolbarDensity="compact"
              after={onTypeFilterChange ? (
                <TargetTypeFacetedFilter
                  title={state.tColumns("common.type")}
                  values={typeFilter ?? []}
                  onValuesChange={onTypeFilterChange}
                  labels={{
                    all: state.tActions("all"),
                    domain: state.tTarget("types.domain"),
                    ip: state.tTarget("types.ip"),
                    cidr: state.tTarget("types.cidr"),
                  }}
                  clearLabel={state.tDataTable("clearFilter")}
                />
              ) : null}
            />
          ),
          className,
          tableClassName,
          toolbarDensity: "compact",
          hideToolbar,
          hidePagination,
          loading,
          loadingPresentation: loading ? "initial" : undefined,
          loadingRowCount,
          stableSurfaceRowCount,
          loadingSlots,
        }}
      />
      <SelectedRowActionBar
        selectedCount={selectedRows.length}
        ariaLabel={state.tDataTable("selected", { count: selectedRows.length })}
        countLabel={state.tDataTable("selected", { count: selectedRows.length })}
        actions={selectedRowActions}
        onClearSelection={onSelectionChange ? () => onSelectionChange([]) : undefined}
        clearSelectionLabel={state.tDataTable("deselectAll")}
      />
    </>
  )
}
