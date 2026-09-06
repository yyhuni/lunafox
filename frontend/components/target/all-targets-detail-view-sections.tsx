"use client"

import dynamic from "next/dynamic"

import { TargetsDataTable } from "@/components/target/targets-data-table"
import { Spinner } from "@/components/shared/loading/spinner"
import {
  AlertDialog,
  AlertDialogClose, AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import {
  deferredInteractionUnmountDelayMs,
  useDeferredInteractionMount,
} from "@/hooks/use-deferred-interaction-mount"

import type { AllTargetsDetailViewState } from "./all-targets-detail-view-state"

const ALL_TARGETS_LOADING_SLOTS = {
  toolbar: "all-targets-toolbar",
  body: "all-targets-table-body",
  pagination: "all-targets-pagination",
}

const AddTargetDialog = dynamic(
  () =>
    import("@/components/target/add-target-dialog").then((mod) => ({
      default: mod.AddTargetDialog,
    })),
  { ssr: false, loading: () => null }
)

const BulkInitiateScanDrawer = dynamic(
  () =>
    import("@/components/scan/initiate-scan-dialog").then((mod) => ({
      default: mod.BulkInitiateScanDrawer,
    })),
  { ssr: false, loading: () => null }
)

const InitiateScanDrawer = dynamic(
  () =>
    import("@/components/scan/initiate-scan-dialog").then((mod) => ({
      default: mod.InitiateScanDrawer,
    })),
  { ssr: false, loading: () => null }
)

const CreateScheduledScanSheet = dynamic(
  () =>
    import("@/components/scan/scheduled/create-scheduled-scan-dialog").then((mod) => ({
      default: mod.CreateScheduledScanSheet,
    })),
  { ssr: false, loading: () => null }
)

export function AllTargetsDetailViewLoadingState({
  state,
  rowCount,
}: {
  state: AllTargetsDetailViewState
  rowCount: number
}) {
  return (
    <TargetsDataTable
      data={[]}
      columns={state.columns}
      onAddNew={state.handleAddTarget}
      onAddHover={() => state.setShouldPrefetchOrgs(true)}
      onBulkDelete={state.handleBatchDelete}
      onBulkInitiateScan={state.handleBulkInitiateScan}
      onSelectionChange={state.setSelectedTargets}
      selectedRows={[]}
      searchPlaceholder={state.tTarget("name")}
      searchValue={state.searchQuery}
      onSearch={state.commitSearch}
      isSearching={state.isSearching}
      addButtonText={state.tTarget("addTarget")}
      pagination={state.pagination}
      paginationNavigation={state.paginationNavigation}
      onPaginationChange={state.handlePaginationChange}
      totalCount={state.totalCount}
      manualPagination={true}
      sorting={state.sorting}
      onSortingChange={state.handleSortingChange}
      typeFilter={state.typeFilter}
      onTypeFilterChange={state.handleTypeFilterChange}
      className={state.className}
      tableClassName={state.tableClassName}
      hideToolbar={state.hideToolbar}
      hidePagination={state.hidePagination}
      loading
      loadingRowCount={rowCount}
      stableSurfaceRowCount={rowCount}
      loadingSlots={ALL_TARGETS_LOADING_SLOTS}
    />
  )
}

export function AllTargetsDetailViewTable({
  state,
  stableSurfaceRowCount,
}: {
  state: AllTargetsDetailViewState
  stableSurfaceRowCount: number
}) {
  return (
    <TargetsDataTable
      data={state.targets}
      columns={state.columns}
      onAddNew={state.handleAddTarget}
      onAddHover={() => state.setShouldPrefetchOrgs(true)}
      onBulkDelete={state.handleBatchDelete}
      onBulkInitiateScan={state.handleBulkInitiateScan}
      onSelectionChange={state.setSelectedTargets}
      selectedRows={state.selectedTargets}
      searchPlaceholder={state.tTarget("name")}
      searchValue={state.searchQuery}
      onSearch={state.commitSearch}
      isSearching={state.isSearching}
      addButtonText={state.tTarget("addTarget")}
      pagination={state.pagination}
      paginationNavigation={state.paginationNavigation}
      onPaginationChange={state.handlePaginationChange}
      totalCount={state.totalCount}
      manualPagination={true}
      sorting={state.sorting}
      onSortingChange={state.handleSortingChange}
      typeFilter={state.typeFilter}
      onTypeFilterChange={state.handleTypeFilterChange}
      className={state.className}
      tableClassName={state.tableClassName}
      hideToolbar={state.hideToolbar}
      hidePagination={state.hidePagination}
      stableSurfaceRowCount={stableSurfaceRowCount}
      loadingSlots={ALL_TARGETS_LOADING_SLOTS}
    />
  )
}

export function AllTargetsDetailViewDialogs({
  state,
}: {
  state: AllTargetsDetailViewState
}) {
  const sidebarMountOptions = { unmountDelayMs: deferredInteractionUnmountDelayMs }
  const shouldMountAddDialog = useDeferredInteractionMount(state.isAddDialogOpen, {
    ...sidebarMountOptions,
    preloadWhen: state.shouldPrefetchOrgs,
  })
  const shouldMountScanDrawer = useDeferredInteractionMount(state.initiateScanDialogOpen, sidebarMountOptions)
  const shouldMountBulkScanDrawer = useDeferredInteractionMount(state.bulkInitiateScanDialogOpen, sidebarMountOptions)
  const shouldMountScheduleSheet = useDeferredInteractionMount(state.scheduleScanDialogOpen, sidebarMountOptions)

  return (
    <>
      {shouldMountAddDialog ? (
        <AddTargetDialog
          onAdd={() => {
            state.setIsAddDialogOpen(false)
          }}
          open={state.isAddDialogOpen}
          onOpenChange={state.setIsAddDialogOpen}
          prefetchEnabled={state.shouldPrefetchOrgs}
        />
      ) : null}

      <AlertDialog open={state.deleteDialogOpen} onOpenChange={state.setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{state.tConfirm("deleteTargetTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {state.tConfirm("deleteTargetMessage", { name: state.targetToDelete?.name ?? "" })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose variant="outline">{state.tCommon("actions.cancel")}</AlertDialogClose>
            <AlertDialogClose
              onClick={state.confirmDelete}
              className="bg-destructive hover:bg-destructive/90 text-destructive-foreground"
              disabled={state.deleteTargetMutation.isPending}
            >
              {state.deleteTargetMutation.isPending ? (
                <>
                  <Spinner />
                  {state.tConfirm("deleting")}
                </>
              ) : (
                state.tConfirm("confirmDelete")
              )}
            </AlertDialogClose>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {shouldMountScanDrawer ? (
        <InitiateScanDrawer
          targetId={state.targetToScan?.id}
          targetName={state.targetToScan?.name}
          open={state.initiateScanDialogOpen}
          onOpenChange={state.setInitiateScanDialogOpen}
          onSuccess={() => {
            state.setTargetToScan(null)
          }}
        />
      ) : null}

      {shouldMountBulkScanDrawer ? (
        <BulkInitiateScanDrawer
          targetIds={state.selectedTargets.map((target) => target.id)}
          scopeLabel={state.tScanInitiate("bulkTargetsDesc", { count: state.selectedTargets.length })}
          open={state.bulkInitiateScanDialogOpen}
          onOpenChange={state.setBulkInitiateScanDialogOpen}
          onSuccess={state.handleBulkInitiateScanSuccess}
        />
      ) : null}

      {shouldMountScheduleSheet ? (
        <CreateScheduledScanSheet
          open={state.scheduleScanDialogOpen}
          onOpenChange={state.setScheduleScanDialogOpen}
          presetTargetId={state.targetToSchedule?.id}
          presetTargetName={state.targetToSchedule?.name}
          onSuccess={() => {
            state.setTargetToSchedule(null)
          }}
        />
      ) : null}

      <AlertDialog open={state.bulkDeleteDialogOpen} onOpenChange={state.setBulkDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{state.tConfirm("bulkDeleteTargetTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {state.tConfirm("bulkDeleteTargetMessage", { count: state.selectedTargets.length })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose variant="outline">{state.tCommon("actions.cancel")}</AlertDialogClose>
            <AlertDialogClose
              onClick={state.confirmBulkDelete}
              className="bg-destructive hover:bg-destructive/90 text-destructive-foreground"
              disabled={state.batchDeleteMutation.isPending}
            >
              {state.batchDeleteMutation.isPending ? (
                <>
                  <Spinner />
                  {state.tConfirm("deleting")}
                </>
              ) : (
                state.tConfirm("confirmDelete")
              )}
            </AlertDialogClose>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
