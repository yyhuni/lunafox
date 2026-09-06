"use client"

import React from "react"
import { useTranslations } from "next-intl"

import { ScanHistoryDialogs } from "@/components/scan/history/scan-history-list-dialogs"
import { ScanProgressDialog } from "@/components/scan/scan-progress-dialog"
import { ScanRuntimeDetailDrawer } from "@/components/scan/history/scan-runtime-detail-drawer"
import { getLoadingOwnerAttributes } from "@/components/shared/loading/loading-owner"
import { ScanHistoryDataTable } from "./scan-history-data-table"
import { createScanHistoryColumns, type ScanHistoryTranslations } from "./scan-history-columns"

import type { LoadingLayer } from "@/components/shared/loading/loading-owner"
import type { ScanHistoryListViewState } from "./scan-history-list-view-state"

const SCAN_HISTORY_LIST_LOADING_SLOTS = {
  toolbar: "scan-history-list-toolbar",
  body: "scan-history-list-body",
  pagination: "scan-history-list-pagination",
}

export function ScanHistoryListLoadingState({
  owner,
  hideToolbar = false,
  hidePagination = false,
  rowCount = 8,
  hideTargetColumn = false,
  layer = "workspace",
}: {
  owner?: string
  hideToolbar?: boolean
  hidePagination?: boolean
  rowCount?: number
  hideTargetColumn?: boolean
  layer?: LoadingLayer
}) {
  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const tTooltips = useTranslations("tooltips")
  const tScan = useTranslations("scan")

  if (owner !== undefined && !owner.trim()) {
    throw new Error("ScanHistoryListLoadingState requires a non-empty owner.")
  }

  const loadingTranslations = {
    columns: {
      target: tColumns("scanHistory.target"),
      summary: tColumns("scanHistory.summary"),
      executedEngines: tColumns("scanHistory.executedEngines"),
      triggerType: tColumns("scanHistory.triggerType"),
      createdAt: tColumns("common.createdAt"),
      status: tColumns("common.status"),
      progress: tColumns("scanHistory.progress"),
    },
    actions: {
      scanDetail: tScan("history.actions.scanDetail"),
      runtimeDetail: tScan("history.runtimeDrawer.open"),
      openMenu: tCommon("actions.openMenu"),
      stop: tCommon("actions.stop"),
      stopScanPending: tScan("stopScanPending"),
      delete: tCommon("actions.delete"),
      selectAll: tCommon("actions.selectAll"),
      selectRow: tCommon("actions.selectRow"),
    },
    tooltips: {
      viewProgress: tTooltips("viewProgress"),
    },
    status: {
      cancelled: tCommon("status.cancelled"),
      succeeded: tCommon("status.succeeded"),
      failed: tCommon("status.failed"),
      pending: tCommon("status.pending"),
      running: tCommon("status.running"),
    },
    summary: {
      subdomains: tColumns("scanHistory.subdomains"),
      websites: tColumns("scanHistory.websites"),
      ipAddresses: tColumns("scanHistory.ipAddresses"),
      endpoints: tColumns("scanHistory.endpoints"),
      vulnerabilities: tColumns("scanHistory.vulnerabilities"),
    },
    triggerTypes: {
      manual: tScan("history.triggerType.manual"),
      scheduled: tScan("history.triggerType.scheduled"),
      ai: tScan("history.triggerType.ai"),
    },
  } satisfies ScanHistoryTranslations

  const columns = createScanHistoryColumns({
    formatDate: (value) => value,
    handleDelete: () => {},
    handleStop: () => {},
    handleStatusClick: () => {},
    statusActionLabel: loadingTranslations.actions.runtimeDetail,
    statusClickable: false,
    t: loadingTranslations,
    hideTargetColumn,
  })

  return (
    <div
      {...(owner ? getLoadingOwnerAttributes({ owner, layer, intent: "data" }) : {})}
      data-slot="scan-history-list-loading-state"
      className="w-full"
    >
      <ScanHistoryDataTable
        data={[]}
        columns={columns}
        onBulkDelete={hideToolbar ? undefined : () => {}}
        searchPlaceholder={tScan("history.searchPlaceholder")}
        searchValue=""
        onSearch={() => {}}
        pagination={{ pageIndex: 0, pageSize: rowCount }}
        cursorPaginationSummary={{ total: 0 }}
        paginationNavigation={{ mode: "cursor", canFirstPage: false, canPreviousPage: false, canNextPage: false }}
        onPaginationChange={() => {}}
        sortingMode="server"
        sorting={[]}
        onSortingChange={() => {}}
        hideToolbar={hideToolbar}
        hidePagination={hidePagination}
        statusFilter={[]}
        onStatusFilterChange={hideToolbar ? undefined : () => {}}
        loading
        initialLoading
        loadingRowCount={rowCount}
        stableSurfaceRowCount={rowCount}
        loadingSlots={SCAN_HISTORY_LIST_LOADING_SLOTS}
      />
    </div>
  )
}

export function ScanHistoryListTable({
  state,
  stableSurfaceRowCount,
}: {
  state: ScanHistoryListViewState
  stableSurfaceRowCount: number
}) {
  return (
    <ScanHistoryDataTable
      data={state.scans}
      columns={state.scanColumns}
      onBulkDelete={state.hideToolbar ? undefined : state.handleBulkDelete}
      bulkDeleteLabel={state.tCommon("actions.delete")}
      selectedRowActions={state.hideToolbar ? undefined : state.selectedRowActions}
      rowSelection={state.rowSelection}
      onRowSelectionChange={state.setRowSelection}
      selectedRows={state.selectedScans}
      onRowClick={state.handleViewRuntimeDetail}
      rowActionLabel={state.tScan("history.runtimeDrawer.open")}
      searchPlaceholder={state.tScan("history.searchPlaceholder")}
      searchValue={state.searchQuery}
      onSearch={state.commitSearch}
      isSearching={state.isSearching}
      pagination={state.pagination}
      cursorPaginationSummary={state.cursorPaginationSummary}
      paginationNavigation={state.paginationNavigation}
      onPaginationChange={state.handlePaginationChange}
      sortingMode={state.sortingMode}
      sorting={state.sorting}
      onSortingChange={state.handleSortingChange}
      hideToolbar={state.hideToolbar}
      pageSizeOptions={state.pageSizeOptions}
      hidePagination={state.hidePagination}
      statusFilter={state.statusFilter}
      onStatusFilterChange={state.handleStatusFilterChange}
      stableSurfaceRowCount={stableSurfaceRowCount}
      loadingSlots={SCAN_HISTORY_LIST_LOADING_SLOTS}
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
        batchStopDialogOpen={state.batchStopDialogOpen}
        setBatchStopDialogOpen={state.setBatchStopDialogOpen}
        activeSelectedCount={state.activeSelectedScans.length}
        terminalSelectedCount={state.terminalSelectedCount}
        onConfirmBatchStop={state.confirmBatchStop}
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
