"use client"

import React from "react"
import dynamic from "next/dynamic"
import { AlertTriangle } from "@/components/icons"

import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"
import { TargetsDataTable } from "./targets/targets-data-table"
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

import type { OrganizationDetailViewState } from "./organization-detail-view-state"

const AddTargetDialog = dynamic(
  () => import("./targets/add-target-dialog").then((mod) => mod.AddTargetDialog),
  { ssr: false }
)

export function OrganizationDetailViewLoadingState() {
  return <DataTableSkeleton toolbarButtonCount={3} rows={6} columns={4} />
}

export function OrganizationDetailViewErrorState({
  error,
  onRetry,
  tCommon,
}: {
  error: Error
  onRetry: () => void
  tCommon: (key: string) => string
}) {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <div className="rounded-full bg-destructive/10 p-3 mb-4">
        <AlertTriangle className="h-10 w-10 text-destructive" />
      </div>
      <h3 className="text-lg font-semibold mb-2">{tCommon("status.loadFailed")}</h3>
      <p className="text-muted-foreground text-center mb-4">{error.message}</p>
      <button type="button"
        onClick={onRetry}
        className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
      >
        {tCommon("actions.reload")}
      </button>
    </div>
  )
}

export function OrganizationDetailViewEmptyState({
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

export function OrganizationDetailViewTable({
  state,
}: {
  state: OrganizationDetailViewState
}) {
  return (
    <TargetsDataTable
      data={state.targetRows}
      columns={state.targetColumns}
      onAddNew={state.handleAddTarget}
      onBulkDelete={state.handleBulkDelete}
      onSelectionChange={state.setSelectedTargets}
      searchPlaceholder={state.tColumns("target.target")}
      searchValue={state.searchQuery}
      onSearch={state.handleSearchChange}
      isSearching={state.isSearching}
      addButtonText={state.tTarget("addTarget")}
      pagination={state.pagination}
      setPagination={state.setPagination}
      paginationInfo={state.paginationInfo}
      onPaginationChange={state.handlePaginationChange}
      typeFilter={state.typeFilter}
      onTypeFilterChange={state.handleTypeFilterChange}
    />
  )
}

export function OrganizationDetailViewDialogs({
  state,
}: {
  state: OrganizationDetailViewState
}) {
  return (
    <>
      {state.isAddDialogOpen ? (
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
            <AlertDialogCancel>{state.tCommon("actions.cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={state.confirmDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {state.tConfirm("confirmUnlink")}
            </AlertDialogAction>
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
            >
              {state.tConfirm("confirmUnlinkCount", { count: state.selectedTargets.length })}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
