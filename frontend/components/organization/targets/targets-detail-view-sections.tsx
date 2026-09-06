"use client"

import dynamic from "next/dynamic"

import { TargetsDataTable } from "./targets-data-table"
import { getDataTableSkeletonRowCount } from "@/components/shared/loading/data-table-skeleton"
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

import type { TargetsDetailViewState } from "./targets-detail-view-state"

const AddTargetDialog = dynamic(
  () =>
    import("./add-target-dialog").then((mod) => ({
      default: mod.AddTargetDialog,
    })),
  { ssr: false, loading: () => null }
)

export function TargetsDetailViewLoadingState({
  state,
}: {
  state: TargetsDetailViewState
}) {
  return (
    <TargetsDataTable
      data={[]}
      columns={state.targetColumns}
      onAddNew={state.handleAddTarget}
      onBulkDelete={state.handleBulkDelete}
      onSelectionChange={state.setSelectedTargets}
      selectedRows={state.selectedTargets}
      addButtonText={state.tTarget("addTarget")}
      pagination={state.pagination}
      cursorPaginationSummary={state.cursorPaginationSummary}
      paginationNavigation={state.paginationNavigation}
      onPaginationChange={state.handlePaginationChange}
      loading
      loadingRowCount={getDataTableSkeletonRowCount(state.pagination.pageSize)}
    />
  )
}

export function TargetsDetailViewEmptyState({
  tOrg,
}: {
  tOrg: (key: string) => string
}) {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <p className="text-muted-foreground">{tOrg("notFound")}</p>
    </div>
  )
}

export function TargetsDetailViewTable({
  state,
}: {
  state: TargetsDetailViewState
}) {
  return (
    <TargetsDataTable
      data={state.targetRows}
      columns={state.targetColumns}
      onAddNew={state.handleAddTarget}
      onBulkDelete={state.handleBulkDelete}
      onSelectionChange={state.setSelectedTargets}
      selectedRows={state.selectedTargets}
      addButtonText={state.tTarget("addTarget")}
      pagination={state.pagination}
      cursorPaginationSummary={state.cursorPaginationSummary}
      paginationNavigation={state.paginationNavigation}
      onPaginationChange={state.handlePaginationChange}
    />
  )
}

export function TargetsDetailViewDialogs({
  state,
}: {
  state: TargetsDetailViewState
}) {
  const shouldMountAddDialog = useDeferredInteractionMount(state.isAddDialogOpen, {
    unmountDelayMs: deferredInteractionUnmountDelayMs,
  })

  return (
    <>
      {shouldMountAddDialog ? (
        <AddTargetDialog
          organizationId={state.organization?.id ?? 0}
          organizationName={state.organization?.name ?? ""}
          onAdd={state.handleAddSuccess}
          open={state.isAddDialogOpen}
          onOpenChange={state.setIsAddDialogOpen}
        />
      ) : null}

      <AlertDialog open={state.deleteDialogOpen} onOpenChange={state.setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{state.tConfirm("unlinkTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {state.tConfirm("unlinkTargetMessage", { name: state.targetToDelete?.name ?? "" })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose variant="outline">{state.tCommon("actions.cancel")}</AlertDialogClose>
            <AlertDialogClose
              onClick={state.confirmDelete}
              className="bg-destructive hover:bg-destructive/90 text-destructive-foreground"
            >
              {state.tConfirm("confirmUnlink")}
            </AlertDialogClose>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={state.bulkDeleteDialogOpen} onOpenChange={state.setBulkDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{state.tConfirm("bulkUnlinkTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {state.tConfirm("bulkUnlinkTargetMessage", { count: state.selectedTargets.length })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose variant="outline">{state.tCommon("actions.cancel")}</AlertDialogClose>
            <AlertDialogClose
              onClick={state.confirmBulkDelete}
              className="bg-destructive hover:bg-destructive/90 text-destructive-foreground"
            >
              {state.tConfirm("confirmUnlink")}
            </AlertDialogClose>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
