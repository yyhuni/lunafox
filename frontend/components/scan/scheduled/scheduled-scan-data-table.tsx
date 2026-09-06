"use client"

import type { ColumnDef } from "@tanstack/react-table"
import type { ReactNode } from "react"

import { UnifiedDataTable } from "@/components/shared/data-table/unified-data-table"
import type { SelectedRowActionBarAction } from "@/components/shared/data-table"
import { SimpleSearchToolbar } from "@/components/shared/data-table/simple-search-toolbar"
import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { SearchToolbarSkeleton } from "@/components/shared/loading/search-toolbar-skeleton"
import { Button } from "@/components/ui/button"
import { Tabs, TabsCountBadge, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Plus } from "@/components/icons"
import { cn } from "@/lib/utils"
import { useScheduledScanDataTableState } from "./scheduled-scan-data-table-state"
import { SCHEDULED_SCAN_PAGE_SIZE } from "@/components/scan/scheduled/scheduled-scan-page-layout"

import type { ScheduledScan } from "@/types/scheduled-scan.types"
import type {
  CursorPaginationNavigation,
  CursorPaginationSummary,
} from "@/types/data-table.types"

export type ScheduledScanQuickFilter =
  | "all"
  | "enabled"
  | "paused"

type ScheduledScanQuickFilterCounts = Record<ScheduledScanQuickFilter, number>

interface ScheduledScanDataTableProps {
  data: ScheduledScan[]
  columns: ColumnDef<ScheduledScan>[]
  onAddNew?: () => void
  onBulkDelete?: () => void
  selectedRowActions?: SelectedRowActionBarAction[]
  onRowClick?: (scan: ScheduledScan) => void
  onSelectionChange?: (selectedRows: ScheduledScan[]) => void
  selectedRows?: ScheduledScan[]
  searchAfter?: ReactNode
  searchPlaceholder?: string
  searchValue?: string
  onSearch?: (value: string) => void
  isSearching?: boolean
  addButtonText?: string
  page?: number
  pageSize?: number
  total?: number
  totalPages?: number
  cursorPaginationSummary?: CursorPaginationSummary
  paginationNavigation?: CursorPaginationNavigation
  onPageChange?: (page: number) => void
  onPageSizeChange?: (pageSize: number) => void
  quickFilter?: ScheduledScanQuickFilter
  onQuickFilterChange?: (filter: ScheduledScanQuickFilter) => void
  quickFilterLabels?: Record<ScheduledScanQuickFilter, string>
  quickFilterCounts?: ScheduledScanQuickFilterCounts
  showQuickFilters?: boolean
  enableRowSelection?: boolean
  loading?: boolean
  initialLoading?: boolean
  loadingRowCount?: number
  loadingRowHeightEstimate?: number | ((rowIndex: number) => number)
  stableSurfaceRowCount?: number
}

const defaultQuickFilterLabels: Record<ScheduledScanQuickFilter, string> = {
  all: "All",
  enabled: "Enabled",
  paused: "Paused",
}

function ScheduledScanFilterTabs({
  filters,
  quickFilter,
  onQuickFilterChange,
  quickFilterLabels,
  quickFilterCounts,
  disabled = false,
}: {
  filters: ScheduledScanQuickFilter[]
  quickFilter: ScheduledScanQuickFilter
  onQuickFilterChange?: (filter: ScheduledScanQuickFilter) => void
  quickFilterLabels: Record<ScheduledScanQuickFilter, string>
  quickFilterCounts?: ScheduledScanQuickFilterCounts
  disabled?: boolean
}) {
  return (
    <Tabs
      value={quickFilter}
      onValueChange={(value) => onQuickFilterChange?.(value as ScheduledScanQuickFilter)}
      className="min-w-0"
    >
      <TabsList variant="filter" className="max-w-full overflow-x-auto" size="sm">
        {filters.map((filter) => {
          const count = quickFilterCounts?.[filter]

          return (
            <TabsTrigger
              key={filter}
              value={filter}
              variant="filter"
              size="sm"
              disabled={disabled}
              aria-label={typeof count === "number" ? `${quickFilterLabels[filter]} ${count}` : undefined}
            >
              <span>{quickFilterLabels[filter]}</span>
              {typeof count === "number" ? (
                <TabsCountBadge>
                  {count}
                </TabsCountBadge>
              ) : null}
            </TabsTrigger>
          )
        })}
      </TabsList>
    </Tabs>
  )
}

function ScheduledScanTableToolbar({
  localSearchValue,
  handleSearchInputChange,
  commitSearch,
  loading,
  searchPlaceholder,
  searchAfter,
  onAddNew,
  addButtonText,
  quickFilter = "all",
  onQuickFilterChange,
  quickFilterLabels = defaultQuickFilterLabels,
  quickFilterCounts,
  showQuickFilters = true,
  initialLoading = false,
}: {
  localSearchValue: string
  handleSearchInputChange: (value: string) => void
  commitSearch?: () => void
  loading: boolean
  searchPlaceholder: string
  searchAfter?: ReactNode
  onAddNew?: () => void
  addButtonText: string
  quickFilter?: ScheduledScanQuickFilter
  onQuickFilterChange?: (filter: ScheduledScanQuickFilter) => void
  quickFilterLabels?: Record<ScheduledScanQuickFilter, string>
  quickFilterCounts?: ScheduledScanQuickFilterCounts
  showQuickFilters?: boolean
  initialLoading?: boolean
}) {
  const quickFilters: ScheduledScanQuickFilter[] = [
    "all",
    "enabled",
    "paused",
  ]

  return (
    <div
      data-slot="scheduled-scan-table-toolbar"
      data-loading-presentation={initialLoading ? "initial" : undefined}
      inert={initialLoading ? true : undefined}
      className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between"
    >
      {showQuickFilters ? (
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <ScheduledScanFilterTabs
            filters={quickFilters}
            quickFilter={quickFilter}
            onQuickFilterChange={onQuickFilterChange}
            quickFilterLabels={quickFilterLabels}
            quickFilterCounts={quickFilterCounts}
            disabled={initialLoading}
          />
        </div>
      ) : null}
      <div
        className={cn(
          "flex flex-wrap items-center gap-2",
          showQuickFilters ? "sm:justify-end" : "w-full justify-between sm:justify-between"
        )}
      >
        {initialLoading ? (
          <SearchToolbarSkeleton toolbarDensity="compact" />
        ) : (
          <SimpleSearchToolbar
            value={localSearchValue}
            onChange={handleSearchInputChange}
            onSubmit={commitSearch}
            loading={loading}
            placeholder={searchPlaceholder}
            toolbarDensity="compact"
            after={searchAfter}
          />
        )}
        {initialLoading ? (
          <ActionSkeleton size="sm" widthClassName="w-24" />
        ) : onAddNew ? (
          <Button type="button" size="sm" onClick={onAddNew}>
            <Plus className="h-4 w-4" />
            {addButtonText}
          </Button>
        ) : null}
      </div>
    </div>
  )
}

/**
 * Scheduled scan data table component
 * Uses UnifiedDataTable unified component
 */
export function ScheduledScanDataTable({
  data = [],
  columns,
  onAddNew,
  onBulkDelete,
  selectedRowActions,
  onRowClick,
  onSelectionChange,
  selectedRows,
  searchAfter,
  searchPlaceholder,
  searchValue,
  onSearch,
  isSearching = false,
  addButtonText,
  page = 1,
  pageSize = SCHEDULED_SCAN_PAGE_SIZE,
  total = 0,
  totalPages = 1,
  cursorPaginationSummary,
  paginationNavigation,
  onPageChange,
  onPageSizeChange,
  quickFilter = "all",
  onQuickFilterChange,
  quickFilterLabels,
  quickFilterCounts,
  showQuickFilters = true,
  enableRowSelection = true,
  loading = false,
  initialLoading = false,
  loadingRowCount,
  loadingRowHeightEstimate,
  stableSurfaceRowCount,
}: ScheduledScanDataTableProps) {
  const state = useScheduledScanDataTableState({
    searchValue,
    onSearch,
    page,
    pageSize,
    total,
    totalPages,
    cursorPaginationSummary,
    paginationNavigation,
  })

  const handlePaginationChange = (newPagination: { pageIndex: number; pageSize: number }) => {
    if (newPagination.pageSize !== pageSize && onPageSizeChange) {
      onPageSizeChange(newPagination.pageSize)
      return
    }
    if (newPagination.pageIndex !== page - 1 && onPageChange) {
      onPageChange(newPagination.pageIndex + 1)
    }
  }

  return (
    <div className="space-y-4">
      <ScheduledScanTableToolbar
        localSearchValue={state.localSearchValue}
        handleSearchInputChange={state.handleSearchInputChange}
        commitSearch={state.commitSearch}
        loading={isSearching}
        searchPlaceholder={searchPlaceholder || state.tScan("searchPlaceholder")}
        searchAfter={searchAfter}
        onAddNew={onAddNew}
        addButtonText={addButtonText || state.tScan("createTitle")}
        quickFilter={quickFilter}
        onQuickFilterChange={onQuickFilterChange}
        quickFilterLabels={quickFilterLabels}
        quickFilterCounts={quickFilterCounts}
        showQuickFilters={showQuickFilters}
        initialLoading={initialLoading}
      />
      <UnifiedDataTable<ScheduledScan>
        data={data}
        columns={columns}
        getRowId={(row) => String(row.id)}
        state={{
          pagination: state.pagination,
          paginationInfo: state.paginationInfo,
          cursorPaginationSummary: state.cursorPaginationSummary,
          paginationNavigation: state.paginationNavigation,
          onPaginationChange: handlePaginationChange,
          onSelectionChange,
          selectedRows,
        }}
        behavior={{
          enableRowSelection,
          onRowClick: onRowClick
            ? (row) => onRowClick(row as ScheduledScan)
            : undefined,
        }}
        actions={{
          selectedRowActions,
          onBulkDelete,
          bulkDeleteLabel: state.tCommon("actions.delete"),
          showBulkDelete: Boolean(onBulkDelete),
          showAddButton: false,
          deleteConfirmation: onBulkDelete
            ? {
                title: state.tConfirm("bulkDeleteScheduledScanTitle"),
                description: (count) => state.tConfirm("bulkDeleteScheduledScanMessage", { count }),
                confirmLabel: state.tConfirm("confirmDelete"),
                cancelLabel: state.tCommon("actions.cancel"),
              }
            : undefined,
        }}
        ui={{
          emptyMessage: state.t("noData"),
          hideToolbar: true,
          loading,
          loadingPresentation: initialLoading ? "initial" : "rows",
          loadingRowCount,
          loadingRowHeightEstimate,
          stableSurfaceRowCount,
        }}
      />
    </div>
  )
}
