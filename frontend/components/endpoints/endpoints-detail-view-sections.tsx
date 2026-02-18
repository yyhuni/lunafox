"use client"

import { AlertTriangle } from "@/components/icons"

import { EndpointsDataTable } from "./endpoints-data-table"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"
import { BulkAddUrlsDialog } from "@/components/common/bulk-add-urls-dialog"
import { ConfirmDialog } from "@/components/ui/confirm-dialog"
import { LoadingSpinner } from "@/components/loading-spinner"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"

import type { TargetType } from "@/lib/url-validator"
import type { EndpointsDetailViewState } from "./endpoints-detail-view-state"

export function EndpointsDetailViewErrorState({ state }: { state: EndpointsDetailViewState }) {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <div className="rounded-full bg-destructive/10 p-3 mb-4">
        <AlertTriangle className="h-10 w-10 text-destructive" />
      </div>
      <h3 className="text-lg font-semibold mb-2">{state.tCommon("status.error")}</h3>
      <p className="text-muted-foreground text-center mb-4">
        {state.error?.message || state.tCommon("status.error")}
      </p>
      <button type="button"
        onClick={() => state.refetch()}
        className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
      >
        {state.tCommon("actions.retry")}
      </button>
    </div>
  )
}

export function EndpointsDetailViewLoadingState() {
  return <DataTableSkeleton toolbarButtonCount={2} rows={6} columns={5} />
}

export function EndpointsDetailViewContent({ state }: { state: EndpointsDetailViewState }) {
  return (
    <EndpointsDataTable
      data={state.data?.endpoints || []}
      columns={state.endpointColumns}
      filterValue={state.filterQuery}
      onFilterChange={state.handleFilterChange}
      isSearching={state.isSearching}
      pagination={state.pagination}
      onPaginationChange={state.handlePaginationChange}
      totalCount={state.paginationInfo.total}
      totalPages={state.paginationInfo.totalPages}
      onSelectionChange={state.handleSelectionChange}
      onDownloadAll={state.handleDownloadAll}
      onDownloadSelected={state.handleDownloadSelected}
      onBulkDelete={state.targetId ? () => state.setBulkDeleteDialogOpen(true) : undefined}
      onBulkAdd={state.targetId ? () => state.setBulkAddDialogOpen(true) : undefined}
    />
  )
}

export function EndpointsDetailViewDialogs({ state }: { state: EndpointsDetailViewState }) {
  return (
    <>
      {state.targetId ? (
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
            <AlertDialogCancel>{state.tCommon("actions.cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={state.confirmDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={state.deleteEndpoint.isPending}
            >
              {state.deleteEndpoint.isPending ? (
                <>
                  <LoadingSpinner />
                  {state.tCommon("status.loading")}
                </>
              ) : (
                state.tCommon("actions.delete")
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
