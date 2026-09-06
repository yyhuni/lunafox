"use client"

import dynamic from "next/dynamic"

import { DirectoriesDataTable } from "./directories-data-table"
import { useDirectoryTableColumns } from "./directories-columns"
import { ConfirmDialog } from "@/components/shared/feedback/confirm-dialog"
import {
  deferredInteractionUnmountDelayMs,
  useDeferredInteractionMount,
} from "@/hooks/use-deferred-interaction-mount"

import type { TargetType } from "@/lib/url-validator"
import type { DirectoriesViewState } from "./directories-view-state"

const noop = () => undefined

const BulkAddUrlsDialog = dynamic(
  () =>
    import("@/components/common/bulk-add-urls-dialog").then((mod) => ({
      default: mod.BulkAddUrlsDialog,
    })),
  { ssr: false, loading: () => null }
)

export function DirectoriesViewLoadingState({
  state,
  rowCount,
}: {
  state: DirectoriesViewState
  rowCount: number
}) {
  return (
    <DirectoriesDataTable
      data={[]}
      columns={state.columns}
      searchValue={state.filterQuery}
      onSearchChange={state.commitFilterSearch}
      isSearching={state.isSearching}
      statusFilter={state.statusFilter}
      onStatusFilterChange={state.handleStatusFilterChange}
      statusOptions={state.statusOptions}
      contentTypeFilter={state.contentTypeFilter}
      onContentTypeFilterChange={state.handleContentTypeFilterChange}
      contentTypeOptions={state.contentTypeOptions}
      pagination={state.pagination}
      cursorPaginationSummary={state.cursorPaginationSummary}
      paginationNavigation={state.paginationNavigation}
      onPaginationChange={state.handlePaginationChange}
      sortingMode={state.sortingMode}
      sorting={state.sorting}
      onSortingChange={state.handleSortingChange}
      onSelectionChange={state.isReadOnly ? undefined : state.handleSelectionChange}
      selectedRows={[]}
      onExportAll={state.isReadOnly ? undefined : state.handleExportAll}
      onExportSelected={state.isReadOnly ? undefined : state.handleExportSelected}
      onBulkDelete={!state.isReadOnly && state.targetId ? () => state.setDeleteDialogOpen(true) : undefined}
      onBulkAdd={!state.isReadOnly && state.targetId ? () => state.setBulkAddDialogOpen(true) : undefined}
      loading
      initialLoading
      loadingRowCount={rowCount}
    />
  )
}

export function DirectoriesViewRouteFallback({
  rowCount,
  totalSize,
  showBulkAdd = false,
}: {
  rowCount: number
  totalSize: number
  showBulkAdd?: boolean
}) {
  const { columns } = useDirectoryTableColumns()

  return (
    <div className="px-4 lg:px-6">
      <DirectoriesDataTable
        data={[]}
        columns={columns}
        pagination={{ pageIndex: 0, pageSize: 10 }}
        paginationNavigation={{ mode: "cursor", canFirstPage: false, canPreviousPage: false, canNextPage: false }}
        cursorPaginationSummary={{ total: totalSize }}
        onPaginationChange={noop}
        sortingMode="server"
        sorting={[{ id: "createdAt", desc: true }]}
        onSortingChange={noop}
        onSelectionChange={noop}
        onExportAll={noop}
        onBulkAdd={showBulkAdd ? noop : undefined}
        loading
        initialLoading
        loadingRowCount={rowCount}
      />
    </div>
  )
}

export function DirectoriesViewContent({ state }: { state: DirectoriesViewState }) {
  const shouldMountBulkAddDialog = useDeferredInteractionMount(state.bulkAddDialogOpen, {
    unmountDelayMs: deferredInteractionUnmountDelayMs,
  })

  return (
    <>
      <DirectoriesDataTable
        data={state.directories}
        columns={state.columns}
        searchValue={state.filterQuery}
        onSearchChange={state.commitFilterSearch}
        isSearching={state.isSearching}
        statusFilter={state.statusFilter}
        onStatusFilterChange={state.handleStatusFilterChange}
        statusOptions={state.statusOptions}
        contentTypeFilter={state.contentTypeFilter}
        onContentTypeFilterChange={state.handleContentTypeFilterChange}
      contentTypeOptions={state.contentTypeOptions}
      pagination={state.pagination}
      cursorPaginationSummary={state.cursorPaginationSummary}
      paginationNavigation={state.paginationNavigation}
      onPaginationChange={state.handlePaginationChange}
        sortingMode={state.sortingMode}
        sorting={state.sorting}
        onSortingChange={state.handleSortingChange}
        onSelectionChange={state.isReadOnly ? undefined : state.handleSelectionChange}
        selectedRows={state.selectedDirectories}
        onExportAll={state.isReadOnly ? undefined : state.handleExportAll}
        onExportSelected={state.isReadOnly ? undefined : state.handleExportSelected}
        onBulkDelete={!state.isReadOnly && state.targetId ? () => state.setDeleteDialogOpen(true) : undefined}
        onBulkAdd={!state.isReadOnly && state.targetId ? () => state.setBulkAddDialogOpen(true) : undefined}
      />

      {!state.isReadOnly && state.targetId && shouldMountBulkAddDialog ? (
        <BulkAddUrlsDialog
          targetId={state.targetId}
          assetType="directory"
          targetName={state.target?.name}
          targetType={state.target?.type as TargetType}
          open={state.bulkAddDialogOpen}
          onOpenChange={state.setBulkAddDialogOpen}
          onSuccess={() => state.refetch()}
        />
      ) : null}

      {!state.isReadOnly ? (
        <ConfirmDialog
          open={state.deleteDialogOpen}
          onOpenChange={state.setDeleteDialogOpen}
          title={state.tCommon("actions.confirmDelete")}
          description={state.tCommon("actions.deleteConfirmMessage", {
            count: state.selectedDirectories.length,
          })}
          onConfirm={state.handleBulkDelete}
          loading={state.isDeleting}
          variant="destructive"
        />
      ) : null}
    </>
  )
}
