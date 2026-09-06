"use client"

import type { ColumnDef } from "@tanstack/react-table"

import { UnifiedDataTable } from "./unified-data-table"

import type {
  CursorPaginationNavigation,
  CursorPaginationSummary,
  UnifiedDataTableActionConfig,
  UnifiedDataTableBehaviorConfig,
  UnifiedDataTableStateConfig,
  UnifiedDataTableUIConfig,
} from "@/types/data-table.types"

type BusinessListDataTableProps<TData> = {
  data: TData[]
  columns: ColumnDef<TData, unknown>[]
  getRowId?: (row: TData, index: number) => string
  paginationNavigation?: CursorPaginationNavigation
  cursorPaginationSummary?: CursorPaginationSummary
  state?: UnifiedDataTableStateConfig<TData>
  actions?: UnifiedDataTableActionConfig
  behavior?: Pick<
    UnifiedDataTableBehaviorConfig,
    | "searchMode"
    | "onSearch"
    | "enableRowSelection"
    | "enableAutoColumnSizing"
    | "expandColumnIds"
    | "onRowClick"
    | "onRowIntent"
    | "getRowActionLabel"
  >
  ui?: Pick<
    UnifiedDataTableUIConfig,
    | "hidePagination"
    | "pageSizeOptions"
    | "hideToolbar"
    | "showColumnVisibility"
    | "rowDensity"
    | "toolbarLeft"
    | "toolbarRight"
    | "searchPlaceholder"
    | "filterFields"
    | "filterExamples"
    | "emptyMessage"
    | "emptyComponent"
    | "loading"
    | "loadingPresentation"
    | "initialLoadingToolbarFilterCount"
    | "loadingRowCount"
    | "loadingRowHeightEstimate"
    | "stableSurfaceRowCount"
    | "loadingSlots"
    | "className"
    | "tableClassName"
  >
}

export function BusinessListDataTable<TData>({
  data,
  columns,
  getRowId,
  paginationNavigation,
  cursorPaginationSummary,
  state,
  actions,
  behavior,
  ui,
}: BusinessListDataTableProps<TData>) {
  const resolvedPaginationNavigation = paginationNavigation ?? state?.paginationNavigation
  const resolvedCursorPaginationSummary = cursorPaginationSummary ?? state?.cursorPaginationSummary
  const resolvedState = state?.paginationInfo && state.sortingMode === undefined
    ? {
        ...state,
        cursorPaginationSummary: resolvedCursorPaginationSummary,
        paginationNavigation: resolvedPaginationNavigation,
        sortingMode: "none" as const,
      }
    : resolvedPaginationNavigation || resolvedCursorPaginationSummary
      ? {
          ...state,
          cursorPaginationSummary: resolvedCursorPaginationSummary,
          paginationNavigation: resolvedPaginationNavigation,
        }
      : state

  return (
    <UnifiedDataTable
      data={data}
      columns={columns}
      getRowId={getRowId}
      state={resolvedState}
      actions={actions}
      behavior={{
        enableAutoColumnSizing: false,
        columnLayout: "fixed",
        ...behavior,
      }}
      ui={{
        toolbarDensity: "compact",
        ...ui,
      }}
    />
  )
}
