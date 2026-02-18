"use client"

import React from "react"

import { TargetsDataTable } from "@/components/target/targets-data-table"
import { AddTargetDialog } from "@/components/target/add-target-dialog"
import { InitiateScanDialog } from "@/components/scan/initiate-scan-dialog"
import { CreateScheduledScanDialog } from "@/components/scan/scheduled/create-scheduled-scan-dialog"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"
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

import type { AllTargetsDetailViewState } from "./all-targets-detail-view-state"

export function AllTargetsDetailViewLoadingState() {
  return <DataTableSkeleton toolbarButtonCount={2} rows={6} columns={5} />
}

export function AllTargetsDetailViewErrorState({
  error,
  tCommon,
}: {
  error: unknown
  tCommon: (key: string) => string
}) {
  return (
    <div className="flex items-center justify-center h-64">
      <div className="text-center">
        <p className="text-destructive mb-2">{tCommon("status.error")}</p>
        <p className="text-sm text-muted-foreground">
          {error instanceof Error ? error.message : String(error)}
        </p>
      </div>
    </div>
  )
}

export function AllTargetsDetailViewTable({
  state,
}: {
  state: AllTargetsDetailViewState
}) {
  return (
    <TargetsDataTable
      data={state.targets}
      columns={state.columns}
      onAddNew={state.handleAddTarget}
      onAddHover={() => state.setShouldPrefetchOrgs(true)}
      onBulkDelete={state.handleBatchDelete}
      onSelectionChange={state.setSelectedTargets}
      searchPlaceholder={state.tTarget("name")}
      searchValue={state.searchQuery}
      onSearch={state.handleSearchChange}
      isSearching={state.isSearching}
      addButtonText={state.tTarget("addTarget")}
      pagination={state.pagination}
      onPaginationChange={state.handlePaginationChange}
      totalCount={state.totalCount}
      manualPagination={true}
      typeFilter={state.typeFilter}
      onTypeFilterChange={state.handleTypeFilterChange}
      className={state.className}
      tableClassName={state.tableClassName}
      hideToolbar={state.hideToolbar}
      hidePagination={state.hidePagination}
    />
  )
}

export function AllTargetsDetailViewDialogs({
  state,
}: {
  state: AllTargetsDetailViewState
}) {
  return (
    <>
      <AddTargetDialog
        onAdd={() => {
          state.setIsAddDialogOpen(false)
        }}
        open={state.isAddDialogOpen}
        onOpenChange={state.setIsAddDialogOpen}
        prefetchEnabled={state.shouldPrefetchOrgs}
      />

      <AlertDialog open={state.deleteDialogOpen} onOpenChange={state.setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{state.tConfirm("deleteTargetTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {state.tConfirm("deleteTargetMessage", { name: state.targetToDelete?.name ?? "" })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{state.tCommon("actions.cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={state.confirmDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={state.deleteTargetMutation.isPending}
            >
              {state.deleteTargetMutation.isPending ? (
                <>
                  <LoadingSpinner />
                  {state.tConfirm("deleting")}
                </>
              ) : (
                state.tConfirm("confirmDelete")
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <InitiateScanDialog
        targetId={state.targetToScan?.id}
        targetName={state.targetToScan?.name}
        open={state.initiateScanDialogOpen}
        onOpenChange={state.setInitiateScanDialogOpen}
        onSuccess={() => {
          state.setTargetToScan(null)
        }}
      />

      <CreateScheduledScanDialog
        open={state.scheduleScanDialogOpen}
        onOpenChange={state.setScheduleScanDialogOpen}
        presetTargetId={state.targetToSchedule?.id}
        presetTargetName={state.targetToSchedule?.name}
        onSuccess={() => {
          state.setTargetToSchedule(null)
        }}
      />

      <AlertDialog open={state.bulkDeleteDialogOpen} onOpenChange={state.setBulkDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{state.tConfirm("bulkDeleteTargetTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {state.tConfirm("bulkDeleteTargetMessage", { count: state.selectedTargets.length })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="mt-2 p-2 bg-muted rounded-md max-h-96 overflow-y-auto">
            <ul className="text-sm space-y-1">
              {state.selectedTargets.map((target) => (
                <li key={target.id} className="flex items-center">
                  <span className="font-medium">{target.name}</span>
                  {target.description ? (
                    <span className="text-muted-foreground ml-2">- {target.description}</span>
                  ) : null}
                </li>
              ))}
            </ul>
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel>{state.tCommon("actions.cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={state.confirmBulkDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={state.batchDeleteMutation.isPending}
            >
              {state.batchDeleteMutation.isPending ? (
                <>
                  <LoadingSpinner />
                  {state.tConfirm("deleting")}
                </>
              ) : (
                state.tConfirm("deleteTargetCount", { count: state.selectedTargets.length })
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
