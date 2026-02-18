"use client"

import { AlertTriangle } from "@/components/icons"

import { WebSitesDataTable } from "./websites-data-table"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"
import { Button } from "@/components/ui/button"
import { BulkAddUrlsDialog } from "@/components/common/bulk-add-urls-dialog"
import { ConfirmDialog } from "@/components/ui/confirm-dialog"

import type { TargetType } from "@/lib/url-validator"
import type { WebSitesViewState } from "./websites-view-state"

export function WebSitesViewErrorState({ state }: { state: WebSitesViewState }) {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <div className="rounded-full bg-destructive/10 p-3 mb-4">
        <AlertTriangle className="h-10 w-10 text-destructive" />
      </div>
      <h3 className="text-lg font-semibold mb-2">{state.tStatus("error")}</h3>
      <p className="text-muted-foreground text-center mb-4">{state.tStatus("error")}</p>
      <Button onClick={() => state.refetch()}>{state.tCommon("actions.retry")}</Button>
    </div>
  )
}

export function WebSitesViewLoadingState() {
  return <DataTableSkeleton toolbarButtonCount={2} rows={6} columns={5} />
}

export function WebSitesViewContent({ state }: { state: WebSitesViewState }) {
  return (
    <>
      <WebSitesDataTable
        data={state.websites}
        columns={state.columns}
        filterValue={state.filterQuery}
        onFilterChange={state.handleFilterChange}
        isSearching={state.isSearching}
        pagination={state.pagination}
        setPagination={state.setPagination}
        paginationInfo={state.paginationInfo}
        onPaginationChange={state.setPagination}
        onSelectionChange={state.handleSelectionChange}
        onDownloadAll={state.handleDownloadAll}
        onDownloadSelected={state.handleDownloadSelected}
        onBulkDelete={state.targetId ? () => state.setDeleteDialogOpen(true) : undefined}
        onBulkAdd={state.targetId ? () => state.setBulkAddDialogOpen(true) : undefined}
      />

      {state.targetId ? (
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
