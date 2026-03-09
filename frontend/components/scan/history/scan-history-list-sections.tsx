"use client"

import React from "react"
import dynamic from "next/dynamic"

import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"
import { ScanHistoryDialogs } from "@/components/scan/history/scan-history-list-dialogs"
import { ScanProgressDialog } from "@/components/scan/scan-progress-dialog"
import { ScanRuntimeDetailDrawer } from "@/components/scan/history/scan-runtime-detail-drawer"

import type { ScanHistoryListViewState } from "./scan-history-list-view-state"

const ScanHistoryDataTable = dynamic(
  () => import("./scan-history-data-table").then((mod) => mod.ScanHistoryDataTable),
  {
    ssr: false,
    loading: () => <DataTableSkeleton rows={6} columns={6} withPadding />,
  }
)

export function ScanHistoryListLoadingState() {
  return <DataTableSkeleton toolbarButtonCount={2} rows={6} columns={6} withPadding={false} />
}

export function ScanHistoryListErrorState({
  state,
}: {
  state: ScanHistoryListViewState
}) {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <p className="text-destructive mb-4">{state.tScan("history.loadFailed")}</p>
      <button type="button"
        onClick={() => {
          void state.refetch()
        }}
        className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
      >
        {state.tScan("history.retry")}
      </button>
    </div>
  )
}

export function ScanHistoryListTable({
  state,
}: {
  state: ScanHistoryListViewState
}) {
  return (
    <ScanHistoryDataTable
      data={state.scans}
      columns={state.scanColumns}
      onBulkDelete={state.hideToolbar ? undefined : state.handleBulkDelete}
      onSelectionChange={state.setSelectedScans}
      searchPlaceholder={state.tScan("history.searchPlaceholder")}
      searchValue={state.searchQuery}
      onSearch={state.handleSearchChange}
      isSearching={state.isSearching}
      pagination={state.pagination}
      setPagination={state.setPagination}
      paginationInfo={state.paginationInfo}
      onPaginationChange={state.handlePaginationChange}
      hideToolbar={state.hideToolbar}
      pageSizeOptions={state.pageSizeOptions}
      hidePagination={state.hidePagination}
      statusFilter={state.statusFilter}
      onStatusFilterChange={state.handleStatusFilterChange}
    />
  )
}

export function ScanHistoryListDialogsSection({
  state,
}: {
  state: ScanHistoryListViewState
}) {
  return (
    <>
      <ScanHistoryDialogs
        tConfirm={state.tConfirm}
        tCommon={state.tCommon}
        deleteDialogOpen={state.deleteDialogOpen}
        setDeleteDialogOpen={state.setDeleteDialogOpen}
        scanToDelete={state.scanToDelete}
        onConfirmDelete={state.confirmDelete}
        bulkDeleteDialogOpen={state.bulkDeleteDialogOpen}
        setBulkDeleteDialogOpen={state.setBulkDeleteDialogOpen}
        selectedScans={state.selectedScans}
        onConfirmBulkDelete={state.confirmBulkDelete}
        stopDialogOpen={state.stopDialogOpen}
        setStopDialogOpen={state.setStopDialogOpen}
        scanToStop={state.scanToStop}
        onConfirmStop={state.confirmStop}
      />

      {state.progressData ? (
        <ScanProgressDialog
          open={state.progressDialogOpen}
          onOpenChange={state.setProgressDialogOpen}
          data={state.progressData}
        />
      ) : null}

      <ScanRuntimeDetailDrawer
        open={state.runtimeDetailOpen}
        onOpenChange={state.setRuntimeDetailOpen}
        scan={state.scanForRuntimeDetail}
      />
    </>
  )
}
