"use client"

import dynamic from "next/dynamic"
import { WebSitesDataTable } from "./websites-data-table"
import { ConfirmDialog } from "@/components/shared/feedback/confirm-dialog"
import { DetailAssetContentFrame } from "@/components/shared/layout/detail-asset-content-frame"
import { useWebSiteTableColumns } from "./websites-columns"
import { DEFAULT_WEBSITE_COLUMN_VISIBILITY } from "./websites-view-state"
import {
  deferredInteractionUnmountDelayMs,
  useDeferredInteractionMount,
} from "@/hooks/use-deferred-interaction-mount"

import type { TargetType } from "@/lib/url-validator"
import type { WebSitesViewState } from "./websites-view-state"
import type { WebSite } from "@/types/website.types"

const noop = () => undefined

const BulkAddUrlsDialog = dynamic(
  () =>
    import("@/components/common/bulk-add-urls-dialog").then((mod) => ({
      default: mod.BulkAddUrlsDialog,
    })),
  { ssr: false, loading: () => null }
)

export function WebSitesViewLoadingState({
  state,
  rowCount,
}: {
  state: WebSitesViewState
  rowCount: number
}) {
  return (
    <WebSitesDataTable
      data={[]}
      columns={state.columns}
      searchValue={state.filterQuery}
      onSearchChange={state.commitFilterSearch}
      isSearching={state.isSearching}
      statusCodeFilter={state.statusCodeFilter}
      statusCodeOptions={state.statusCodeOptions}
      onStatusCodeFilterChange={state.handleStatusCodeFilterChange}
      techFilter={state.techFilter}
      techOptions={state.techOptions}
      onTechFilterChange={state.handleTechFilterChange}
      webserverFilter={state.webserverFilter}
      webserverOptions={state.webserverOptions}
      onWebserverFilterChange={state.handleWebserverFilterChange}
      contentTypeFilter={state.contentTypeFilter}
      contentTypeOptions={state.contentTypeOptions}
      onContentTypeFilterChange={state.handleContentTypeFilterChange}
      vhostFilter={state.vhostFilter}
      vhostOptions={state.vhostOptions}
      onVhostFilterChange={state.handleVhostFilterChange}
      pagination={state.pagination}
      cursorPaginationSummary={state.cursorPaginationSummary}
      paginationNavigation={state.paginationNavigation}
      onPaginationChange={state.handlePaginationChange}
      sortingMode={state.sortingMode}
      sorting={state.sorting}
      onSortingChange={state.handleSortingChange}
      onSelectionChange={state.handleSelectionChange}
      selectedRows={[]}
      columnVisibility={state.columnVisibility}
      onColumnVisibilityChange={state.setColumnVisibility}
      onExportAll={state.handleExportAll}
      onExportSelected={state.handleExportSelected}
      onBulkDelete={state.targetId ? () => state.setDeleteDialogOpen(true) : undefined}
      onBulkAdd={state.targetId ? () => state.setBulkAddDialogOpen(true) : undefined}
      loading
      loadingRowCount={rowCount}
    />
  )
}

export function WebSitesViewRouteFallback({
  rowCount,
  totalSize,
  showBulkAdd = false,
}: {
  rowCount: number
  totalSize: number
  showBulkAdd?: boolean
}) {
  const { columns } = useWebSiteTableColumns()

  return (
    <DetailAssetContentFrame>
      <WebSitesDataTable
        data={[]}
        columns={columns}
        pagination={{ pageIndex: 0, pageSize: 10 }}
        paginationNavigation={{ mode: "cursor", canFirstPage: false, canPreviousPage: false, canNextPage: false }}
        cursorPaginationSummary={{ total: totalSize }}
        onPaginationChange={noop}
        columnVisibility={DEFAULT_WEBSITE_COLUMN_VISIBILITY}
        onColumnVisibilityChange={noop}
        onExportAll={noop}
        onBulkAdd={showBulkAdd ? noop : undefined}
        loading
        initialLoading
        loadingRowCount={rowCount}
      />
    </DetailAssetContentFrame>
  )
}

export function WebSitesViewContent({
  state,
  onRowClick,
}: {
  state: WebSitesViewState
  onRowClick?: (website: WebSite) => void
}) {
  return (
    <>
      <WebSitesDataTable
        data={state.websites}
        columns={state.columns}
        searchValue={state.filterQuery}
        onSearchChange={state.commitFilterSearch}
        isSearching={state.isSearching}
        statusCodeFilter={state.statusCodeFilter}
        statusCodeOptions={state.statusCodeOptions}
        onStatusCodeFilterChange={state.handleStatusCodeFilterChange}
        techFilter={state.techFilter}
        techOptions={state.techOptions}
        onTechFilterChange={state.handleTechFilterChange}
        webserverFilter={state.webserverFilter}
        webserverOptions={state.webserverOptions}
        onWebserverFilterChange={state.handleWebserverFilterChange}
        contentTypeFilter={state.contentTypeFilter}
        contentTypeOptions={state.contentTypeOptions}
        onContentTypeFilterChange={state.handleContentTypeFilterChange}
        vhostFilter={state.vhostFilter}
        vhostOptions={state.vhostOptions}
        onVhostFilterChange={state.handleVhostFilterChange}
        pagination={state.pagination}
        cursorPaginationSummary={state.cursorPaginationSummary}
        paginationNavigation={state.paginationNavigation}
        onPaginationChange={state.handlePaginationChange}
        sortingMode={state.sortingMode}
        sorting={state.sorting}
        onSortingChange={state.handleSortingChange}
        onSelectionChange={state.handleSelectionChange}
        selectedRows={state.selectedWebSites}
        columnVisibility={state.columnVisibility}
        onColumnVisibilityChange={state.setColumnVisibility}
        onExportAll={state.handleExportAll}
        onExportSelected={state.handleExportSelected}
        onBulkDelete={state.targetId ? () => state.setDeleteDialogOpen(true) : undefined}
        onBulkAdd={state.targetId ? () => state.setBulkAddDialogOpen(true) : undefined}
        onRowClick={onRowClick}
      />

      <WebSitesManagementOverlays state={state} />
    </>
  )
}

export function WebSitesManagementOverlays({
  state,
}: {
  state: WebSitesViewState
}) {
  const shouldMountBulkAddDialog = useDeferredInteractionMount(state.bulkAddDialogOpen, {
    unmountDelayMs: deferredInteractionUnmountDelayMs,
  })

  return (
    <>
      {state.targetId && shouldMountBulkAddDialog ? (
        <BulkAddUrlsDialog
          targetId={state.targetId}
          assetType="website"
          targetName={state.target?.name}
          targetType={state.target?.type as TargetType}
          open={state.bulkAddDialogOpen}
          onOpenChange={state.setBulkAddDialogOpen}
          onSuccess={() => state.refetch()}
        />
      ) : null}

      <ConfirmDialog
        open={state.deleteDialogOpen}
        onOpenChange={state.setDeleteDialogOpen}
        title={state.tCommon("actions.confirmDelete")}
        description={state.tCommon("actions.deleteConfirmMessage", {
          count: state.selectedWebSites.length,
        })}
        onConfirm={state.handleBulkDelete}
        loading={state.isDeleting}
        variant="destructive"
      />
    </>
  )
}
