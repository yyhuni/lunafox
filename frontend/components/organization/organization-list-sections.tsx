"use client"

import React from "react"
import { Trash2 } from "@/components/icons"

import { Button } from "@/components/ui/button"
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
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"
import { LoadingSpinner } from "@/components/loading-spinner"

import { OrganizationDataTable } from "./organization-data-table"
import { AddOrganizationDialog } from "./add-organization-dialog"
import { EditOrganizationDialog } from "./edit-organization-dialog"
import { InitiateScanDialog } from "@/components/scan/initiate-scan-dialog"
import { CreateScheduledScanDialog } from "@/components/scan/scheduled/create-scheduled-scan-dialog"

import type { OrganizationListState } from "./organization-list-state"

export function OrganizationListSkeleton() {
  return <DataTableSkeleton toolbarButtonCount={2} rows={6} columns={4} />
}

export function OrganizationListErrorState({
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
        <Trash2 className="text-destructive" />
      </div>
      <h3 className="text-lg font-semibold mb-2">{tCommon("status.error")}</h3>
      <p className="text-muted-foreground text-center mb-4">{error.message}</p>
      <Button variant="outline" onClick={onRetry}>
        {tCommon("actions.retry")}
      </Button>
    </div>
  )
}

export function OrganizationListTable({
  state,
}: {
  state: OrganizationListState
}) {
  return (
    <OrganizationDataTable
      data={state.organizations}
      columns={state.columns}
      onAddNew={() => state.setAddDialogOpen(true)}
      onBulkDelete={state.handleBulkDelete}
      onSelectionChange={state.setSelectedOrganizations}
      searchPlaceholder={state.tOrg("name")}
      searchValue={state.searchQuery}
      onSearch={state.handleSearchChange}
      isSearching={state.isSearching}
      pagination={state.pagination}
      setPagination={state.setPagination}
      paginationInfo={state.paginationInfo}
      onPaginationChange={state.handlePaginationChange}
    />
  )
}

export function OrganizationListDialogs({
  state,
}: {
  state: OrganizationListState
}) {
  return (
    <>
      <AlertDialog open={state.deleteDialogOpen} onOpenChange={state.setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{state.tConfirm("deleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {state.tConfirm("deleteOrgMessage", { name: state.organizationToDelete?.name ?? "" })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{state.tCommon("actions.cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={state.confirmDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={state.deleteOrganization.isPending}
            >
              {state.deleteOrganization.isPending ? (
                <>
                  <LoadingSpinner />
                  {state.tConfirm("deleting")}
                </>
              ) : (
                state.tCommon("actions.delete")
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {state.organizationToEdit ? (
        <EditOrganizationDialog
          organization={state.organizationToEdit}
          open={state.editDialogOpen}
          onOpenChange={state.setEditDialogOpen}
          onEdit={state.handleOrganizationEdited}
        />
      ) : null}

      <AlertDialog open={state.bulkDeleteDialogOpen} onOpenChange={state.setBulkDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{state.tConfirm("bulkDeleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {state.tConfirm("bulkDeleteOrgMessage", { count: state.selectedOrganizations.length })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="mt-2 p-2 bg-muted rounded-md max-h-96 overflow-y-auto">
            <ul className="text-sm space-y-1">
              {state.selectedOrganizations.map((org) => (
                <li key={org.id} className="flex items-center">
                  <span className="font-medium">{org.name}</span>
                  {org.description ? (
                    <span className="ml-2 text-muted-foreground">- {org.description}</span>
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
              disabled={state.batchDeleteOrganizations.isPending}
            >
              {state.batchDeleteOrganizations.isPending ? (
                <>
                  <LoadingSpinner />
                  {state.tConfirm("deleting")}
                </>
              ) : (
                state.tConfirm("deleteOrgCount", { count: state.selectedOrganizations.length })
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AddOrganizationDialog
        open={state.addDialogOpen}
        onOpenChange={state.setAddDialogOpen}
        onAdd={() => {
          state.setAddDialogOpen(false)
        }}
      />

      <InitiateScanDialog
        organization={state.organizationToScan}
        organizationId={state.organizationToScan?.id}
        open={state.initiateScanDialogOpen}
        onOpenChange={state.setInitiateScanDialogOpen}
        onSuccess={() => {
          state.setOrganizationToScan(null)
        }}
      />

      <CreateScheduledScanDialog
        open={state.scheduleScanDialogOpen}
        onOpenChange={state.setScheduleScanDialogOpen}
        presetOrganizationId={state.organizationToSchedule?.id}
        presetOrganizationName={state.organizationToSchedule?.name}
        onSuccess={() => {
          state.setOrganizationToSchedule(null)
        }}
      />
    </>
  )
}
