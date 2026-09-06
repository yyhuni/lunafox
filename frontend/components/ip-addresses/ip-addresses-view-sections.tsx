"use client"

import { useTranslations } from "next-intl"

import { IPAddressesDataTable } from "./ip-addresses-data-table"
import { createIPAddressColumns } from "./ip-addresses-columns"
import { ConfirmDialog } from "@/components/shared/feedback/confirm-dialog"
import { DetailAssetContentFrame } from "@/components/shared/layout/detail-asset-content-frame"

import type { IPAddressesViewState } from "./ip-addresses-view-state"

const IP_ADDRESSES_ROUTE_FALLBACK_PAGE_SIZE = 10

export function IPAddressesViewLoadingState({
  state,
  rowCount,
}: {
  state: IPAddressesViewState
  rowCount: number
}) {
  return (
    <IPAddressesDataTable
      data={[]}
      columns={state.columns}
      filterValue={state.filterQuery}
      onFilterChange={state.commitFilterSearch}
      searchPlaceholder={state.searchPlaceholder}
      portFilter={state.portFilter}
      portOptions={state.portOptions}
      onPortFilterChange={state.handlePortFilterChange}
      pagination={state.pagination}
      cursorPaginationSummary={state.cursorPaginationSummary}
      paginationNavigation={state.paginationNavigation}
      onPaginationChange={state.handlePaginationChange}
      sortingMode={state.sortingMode}
      sorting={state.sorting}
      onSortingChange={state.handleSortingChange}
      onSelectionChange={state.isReadOnly ? undefined : state.handleSelectionChange}
      selectedRows={[]}
      onExportAll={state.isReadOnly ? undefined : state.handleExportAll}
      onExportSelected={state.isReadOnly ? undefined : state.handleExportSelected}
      onBulkDelete={!state.isReadOnly && state.targetId ? () => state.setDeleteDialogOpen(true) : undefined}
      loading
      initialLoading
      loadingRowCount={rowCount}
    />
  )
}

export function IPAddressesViewRouteFallback({ rowCount }: { rowCount: number }) {
  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const tTooltips = useTranslations("tooltips")
  const columns = createIPAddressColumns({
    formatDate: (value) => value,
    t: {
      columns: {
        ipAddress: tColumns("ipAddress.ipAddress"),
        hosts: tColumns("ipAddress.hosts"),
        createdAt: tColumns("common.createdAt"),
        openPorts: tColumns("ipAddress.openPorts"),
      },
      actions: {
        selectAll: tCommon("actions.selectAll"),
        selectRow: tCommon("actions.selectRow"),
      },
      tooltips: {
        allHosts: tTooltips("allHosts"),
        allOpenPorts: tTooltips("allOpenPorts"),
      },
    },
  })

  return (
    <DetailAssetContentFrame>
      <IPAddressesDataTable
        data={[]}
        columns={columns}
        filterValue=""
        onFilterChange={() => {}}
        portFilter={[]}
        portOptions={[]}
        onPortFilterChange={() => {}}
        pagination={{ pageIndex: 0, pageSize: IP_ADDRESSES_ROUTE_FALLBACK_PAGE_SIZE }}
        paginationNavigation={{ mode: "cursor", canFirstPage: false, canPreviousPage: false, canNextPage: false }}
        cursorPaginationSummary={{ total: 0 }}
        onPaginationChange={() => {}}
        sortingMode="server"
        sorting={[]}
        onSortingChange={() => {}}
        onSelectionChange={() => {}}
        selectedRows={[]}
        onExportAll={() => {}}
        onExportSelected={() => {}}
        loading
        initialLoading
        loadingRowCount={rowCount}
      />
    </DetailAssetContentFrame>
  )
}

export function IPAddressesViewContent({ state }: { state: IPAddressesViewState }) {
  return (
    <>
      <IPAddressesDataTable
        data={state.ipAddresses}
        columns={state.columns}
        filterValue={state.filterQuery}
        onFilterChange={state.commitFilterSearch}
        searchPlaceholder={state.searchPlaceholder}
        portFilter={state.portFilter}
        portOptions={state.portOptions}
        onPortFilterChange={state.handlePortFilterChange}
        pagination={state.pagination}
        cursorPaginationSummary={state.cursorPaginationSummary}
        paginationNavigation={state.paginationNavigation}
        onPaginationChange={state.handlePaginationChange}
        sortingMode={state.sortingMode}
        sorting={state.sorting}
        onSortingChange={state.handleSortingChange}
        onSelectionChange={state.isReadOnly ? undefined : state.handleSelectionChange}
        selectedRows={state.selectedIPAddresses}
        onExportAll={state.isReadOnly ? undefined : state.handleExportAll}
        onExportSelected={state.isReadOnly ? undefined : state.handleExportSelected}
        onBulkDelete={!state.isReadOnly && state.targetId ? () => state.setDeleteDialogOpen(true) : undefined}
      />

      {!state.isReadOnly ? (
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
      ) : null}
    </>
  )
}
