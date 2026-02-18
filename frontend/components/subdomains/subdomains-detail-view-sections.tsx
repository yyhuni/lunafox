"use client"

import { AlertTriangle } from "@/components/icons"

import { SubdomainsDataTable } from "./subdomains-data-table"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"
import { BulkAddSubdomainsDialog } from "./bulk-add-subdomains-dialog"
import { ConfirmDialog } from "@/components/ui/confirm-dialog"

import type { SubdomainsDetailViewState } from "./subdomains-detail-view-state"

export function SubdomainsDetailViewErrorState({ state }: { state: SubdomainsDetailViewState }) {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <div className="rounded-full bg-destructive/10 p-3 mb-4">
        <AlertTriangle className="h-10 w-10 text-destructive" />
      </div>
      <h3 className="text-lg font-semibold mb-2">{state.tSubdomains("loadFailed")}</h3>
      <p className="text-muted-foreground text-center mb-4">
        {state.error?.message || state.tSubdomains("loadError")}
      </p>
      <button type="button"
        onClick={() => state.refetch()}
        className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
      >
        {state.tSubdomains("reload")}
      </button>
    </div>
  )
}

export function SubdomainsDetailViewLoadingState() {
  return <DataTableSkeleton toolbarButtonCount={2} rows={6} columns={5} />
}

export function SubdomainsDetailViewContent({ state }: { state: SubdomainsDetailViewState }) {
  return (
    <SubdomainsDataTable
      data={state.subdomains}
      columns={state.subdomainColumns}
      onSelectionChange={state.setSelectedSubdomains}
      filterValue={state.filterQuery}
      onFilterChange={state.handleFilterChange}
      isSearching={state.isSearching}
      onDownloadAll={state.handleDownloadAll}
      onDownloadSelected={state.handleDownloadSelected}
      onBulkDelete={state.targetId ? () => state.setDeleteDialogOpen(true) : undefined}
      pagination={state.pagination}
      setPagination={state.setPagination}
      paginationInfo={state.paginationInfo}
      onPaginationChange={state.handlePaginationChange}
      onBulkAdd={state.targetId ? () => state.setBulkAddOpen(true) : undefined}
    />
  )
}

export function SubdomainsDetailViewDialogs({ state }: { state: SubdomainsDetailViewState }) {
  return (
    <>
      {state.targetId ? (
        <BulkAddSubdomainsDialog
          targetId={state.targetId}
          targetName={state.targetData?.name}
          open={state.bulkAddOpen}
          onOpenChange={state.setBulkAddOpen}
          onSuccess={() => state.refetch()}
        />
      ) : null}

      <ConfirmDialog
        open={state.deleteDialogOpen}
        onOpenChange={state.setDeleteDialogOpen}
        title={state.tCommon("actions.confirmDelete")}
        description={state.tCommon("actions.deleteConfirmMessage", {
          count: state.selectedSubdomains.length,
        })}
        onConfirm={state.handleBulkDelete}
        loading={state.isDeleting}
        variant="destructive"
      />
    </>
  )
}
