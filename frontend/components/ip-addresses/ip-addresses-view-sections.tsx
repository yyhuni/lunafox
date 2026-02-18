"use client"

import { AlertTriangle } from "@/components/icons"

import { IPAddressesDataTable } from "./ip-addresses-data-table"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"
import { Button } from "@/components/ui/button"
import { ConfirmDialog } from "@/components/ui/confirm-dialog"

import type { IPAddressesViewState } from "./ip-addresses-view-state"

export function IPAddressesViewErrorState({ state }: { state: IPAddressesViewState }) {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <div className="rounded-full bg-destructive/10 p-3 mb-4">
        <AlertTriangle className="h-10 w-10 text-destructive" />
      </div>
      <h3 className="text-lg font-semibold mb-2">{state.tStatus("error")}</h3>
      <p className="text-muted-foreground text-center mb-4">
        {state.error?.message || state.tStatus("error")}
      </p>
      <Button onClick={() => state.refetch()}>{state.tCommon("actions.retry")}</Button>
    </div>
  )
}

export function IPAddressesViewLoadingState() {
  return <DataTableSkeleton toolbarButtonCount={1} rows={6} columns={4} />
}

export function IPAddressesViewContent({ state }: { state: IPAddressesViewState }) {
  return (
    <>
      <IPAddressesDataTable
        data={state.ipAddresses}
        columns={state.columns}
        filterValue={state.filterQuery}
        onFilterChange={state.handleFilterChange}
        pagination={state.pagination}
        setPagination={state.setPagination}
        paginationInfo={state.paginationInfo}
        onSelectionChange={state.handleSelectionChange}
        onDownloadAll={state.handleDownloadAll}
        onDownloadSelected={state.handleDownloadSelected}
        onBulkDelete={state.targetId ? () => state.setDeleteDialogOpen(true) : undefined}
      />

      <ConfirmDialog
        open={state.deleteDialogOpen}
        onOpenChange={state.setDeleteDialogOpen}
        title={state.tCommon("actions.confirmDelete")}
        description={state.tCommon("actions.deleteConfirmMessage", {
          count: state.selectedIPAddresses.length,
        })}
        onConfirm={state.handleBulkDelete}
        loading={state.isDeleting}
        variant="destructive"
      />
    </>
  )
}
