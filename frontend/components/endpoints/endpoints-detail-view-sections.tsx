"use client"

import dynamic from "next/dynamic"
import { useTranslations } from "next-intl"

import { EndpointsDataTable } from "./endpoints-data-table"
import { createEndpointColumns } from "./endpoints-columns"
import { DEFAULT_ENDPOINT_COLUMN_VISIBILITY } from "./endpoints-detail-view-state"
import { ConfirmDialog } from "@/components/shared/feedback/confirm-dialog"
import { DetailAssetContentFrame } from "@/components/shared/layout/detail-asset-content-frame"
import { Spinner } from "@/components/shared/loading/spinner"
import {
  deferredInteractionUnmountDelayMs,
  useDeferredInteractionMount,
} from "@/hooks/use-deferred-interaction-mount"
import {
  AlertDialog,
  AlertDialogClose, AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"

import type { TargetType } from "@/lib/url-validator"
import type { EndpointsDetailViewState } from "./endpoints-detail-view-state"
import type { Endpoint } from "@/types/endpoint.types"

const ENDPOINTS_ROUTE_FALLBACK_PAGE_SIZE = 10

const BulkAddUrlsDialog = dynamic(
  () =>
    import("@/components/common/bulk-add-urls-dialog").then((mod) => ({
      default: mod.BulkAddUrlsDialog,
    })),
  { ssr: false, loading: () => null }
)

export function EndpointsDetailViewLoadingState({
  state,
  rowCount,
}: {
  state: EndpointsDetailViewState
  rowCount: number
}) {
  return (
    <EndpointsDataTable
      data={[]}
      columns={state.endpointColumns}
      searchValue={state.filterQuery}
      onSearchChange={state.commitFilterSearch}
      isSearching={state.isSearching}
      statusCodeFilter={state.statusCodeFilter}
      onStatusCodeFilterChange={state.handleStatusCodeFilterChange}
      statusCodeOptions={state.statusCodeOptions}
      techFilter={state.techFilter}
      onTechFilterChange={state.handleTechFilterChange}
      techOptions={state.techOptions}
      webserverFilter={state.webserverFilter}
      onWebserverFilterChange={state.handleWebserverFilterChange}
      webserverOptions={state.webserverOptions}
      contentTypeFilter={state.contentTypeFilter}
      onContentTypeFilterChange={state.handleContentTypeFilterChange}
      contentTypeOptions={state.contentTypeOptions}
      vhostFilter={state.vhostFilter}
      onVhostFilterChange={state.handleVhostFilterChange}
      vhostOptions={state.vhostOptions}
      pagination={state.pagination}
      cursorPaginationSummary={state.cursorPaginationSummary}
      paginationNavigation={state.paginationNavigation}
      onPaginationChange={state.handlePaginationChange}
      sortingMode={state.sortingMode}
      sorting={state.sorting}
      onSortingChange={state.handleSortingChange}
      onSelectionChange={state.isReadOnly ? undefined : state.handleSelectionChange}
      selectedRows={[]}
      columnVisibility={state.columnVisibility}
      onColumnVisibilityChange={state.setColumnVisibility}
      onExportAll={state.isReadOnly ? undefined : state.handleExportAll}
      onExportSelected={state.isReadOnly ? undefined : state.handleExportSelected}
      onBulkDelete={!state.isReadOnly && state.targetId ? () => state.setBulkDeleteDialogOpen(true) : undefined}
      onBulkAdd={!state.isReadOnly && state.targetId ? () => state.setBulkAddDialogOpen(true) : undefined}
      loading
      initialLoading
      loadingRowCount={rowCount}
    />
  )
}

export function EndpointsDetailViewRouteFallback({ rowCount }: { rowCount: number }) {
  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const columns = createEndpointColumns({
    formatDate: (value) => value,
    t: {
      columns: {
        url: tColumns("common.url"),
        host: tColumns("endpoint.host"),
        title: tColumns("endpoint.title"),
        status: tColumns("common.status"),
        contentLength: tColumns("endpoint.contentLength"),
        location: tColumns("endpoint.location"),
        webServer: tColumns("endpoint.webServer"),
        contentType: tColumns("endpoint.contentType"),
        technologies: tColumns("endpoint.technologies"),
        responseBody: tColumns("endpoint.responseBody"),
        vhost: tColumns("endpoint.vhost"),
        responseHeaders: tColumns("endpoint.responseHeaders"),
        createdAt: tColumns("common.createdAt"),
      },
      actions: {
        selectAll: tCommon("actions.selectAll"),
        selectRow: tCommon("actions.selectRow"),
      },
    },
  })

  return (
    <DetailAssetContentFrame>
      <EndpointsDataTable
        data={[]}
        columns={columns}
        searchValue=""
        onSearchChange={() => {}}
        statusCodeFilter={[]}
        onStatusCodeFilterChange={() => {}}
        statusCodeOptions={[]}
        techFilter={[]}
        onTechFilterChange={() => {}}
        techOptions={[]}
        webserverFilter={[]}
        onWebserverFilterChange={() => {}}
        webserverOptions={[]}
        contentTypeFilter={[]}
        onContentTypeFilterChange={() => {}}
        contentTypeOptions={[]}
        vhostFilter={[]}
        onVhostFilterChange={() => {}}
        pagination={{ pageIndex: 0, pageSize: ENDPOINTS_ROUTE_FALLBACK_PAGE_SIZE }}
        paginationNavigation={{ mode: "cursor", canFirstPage: false, canPreviousPage: false, canNextPage: false }}
        cursorPaginationSummary={{ total: 0 }}
        onPaginationChange={() => {}}
        sortingMode="server"
        sorting={[]}
        onSortingChange={() => {}}
        onSelectionChange={() => {}}
        selectedRows={[]}
        columnVisibility={DEFAULT_ENDPOINT_COLUMN_VISIBILITY}
        onColumnVisibilityChange={() => {}}
        onExportAll={() => {}}
        onExportSelected={() => {}}
        onBulkAdd={() => {}}
        loading
        initialLoading
        loadingRowCount={rowCount}
      />
    </DetailAssetContentFrame>
  )
}

export function EndpointsDetailViewContent({
  state,
  onRowClick,
}: {
  state: EndpointsDetailViewState
  onRowClick?: (endpoint: Endpoint) => void
}) {
  return (
    <EndpointsDataTable
      data={state.data?.results || []}
      columns={state.endpointColumns}
      searchValue={state.filterQuery}
      onSearchChange={state.commitFilterSearch}
      isSearching={state.isSearching}
      statusCodeFilter={state.statusCodeFilter}
      onStatusCodeFilterChange={state.handleStatusCodeFilterChange}
      statusCodeOptions={state.statusCodeOptions}
      techFilter={state.techFilter}
      onTechFilterChange={state.handleTechFilterChange}
      techOptions={state.techOptions}
      webserverFilter={state.webserverFilter}
      onWebserverFilterChange={state.handleWebserverFilterChange}
      webserverOptions={state.webserverOptions}
      contentTypeFilter={state.contentTypeFilter}
      onContentTypeFilterChange={state.handleContentTypeFilterChange}
      contentTypeOptions={state.contentTypeOptions}
      vhostFilter={state.vhostFilter}
      onVhostFilterChange={state.handleVhostFilterChange}
      vhostOptions={state.vhostOptions}
      pagination={state.pagination}
      cursorPaginationSummary={state.cursorPaginationSummary}
      paginationNavigation={state.paginationNavigation}
      onPaginationChange={state.handlePaginationChange}
      sortingMode={state.sortingMode}
      sorting={state.sorting}
      onSortingChange={state.handleSortingChange}
      onSelectionChange={state.isReadOnly ? undefined : state.handleSelectionChange}
      selectedRows={state.selectedEndpoints}
      columnVisibility={state.columnVisibility}
      onColumnVisibilityChange={state.setColumnVisibility}
      onExportAll={state.isReadOnly ? undefined : state.handleExportAll}
      onExportSelected={state.isReadOnly ? undefined : state.handleExportSelected}
      onBulkDelete={!state.isReadOnly && state.targetId ? () => state.setBulkDeleteDialogOpen(true) : undefined}
      onBulkAdd={!state.isReadOnly && state.targetId ? () => state.setBulkAddDialogOpen(true) : undefined}
      onRowClick={onRowClick}
    />
  )
}

export function EndpointsDetailViewDialogs({ state }: { state: EndpointsDetailViewState }) {
  const shouldMountBulkAddDialog = useDeferredInteractionMount(state.bulkAddDialogOpen, {
    unmountDelayMs: deferredInteractionUnmountDelayMs,
  })

  if (state.isReadOnly) return null

  return (
    <>
      {state.targetId && shouldMountBulkAddDialog ? (
        <BulkAddUrlsDialog
          targetId={state.targetId}
          assetType="endpoint"
          targetName={state.target?.name}
          targetType={state.target?.type as TargetType}
          open={state.bulkAddDialogOpen}
          onOpenChange={state.setBulkAddDialogOpen}
          onSuccess={() => state.refetch()}
        />
      ) : null}

      <ConfirmDialog
        open={state.bulkDeleteDialogOpen}
        onOpenChange={state.setBulkDeleteDialogOpen}
        title={state.tConfirm("deleteTitle")}
        description={state.tCommon("actions.deleteConfirmMessage", {
          count: state.selectedEndpoints.length,
        })}
        onConfirm={state.handleBulkDelete}
        loading={state.isDeleting}
        variant="destructive"
      />

      <AlertDialog open={state.deleteDialogOpen} onOpenChange={state.setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{state.tConfirm("deleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>{state.tConfirm("deleteMessage")}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose variant="outline">{state.tCommon("actions.cancel")}</AlertDialogClose>
            <AlertDialogClose
              onClick={state.confirmDelete}
              className="bg-destructive hover:bg-destructive/90 text-destructive-foreground"
              disabled={state.deleteEndpoint.isPending}
            >
              {state.deleteEndpoint.isPending ? (
                <>
                  <Spinner />
                  {state.tCommon("status.loading")}
                </>
              ) : (
                state.tCommon("actions.delete")
              )}
            </AlertDialogClose>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
