"use client"

import dynamic from "next/dynamic"
import { useTranslations } from "next-intl"

import { SubdomainsDataTable } from "./subdomains-data-table"
import { createSubdomainColumns } from "./subdomains-columns"
import { ConfirmDialog } from "@/components/shared/feedback/confirm-dialog"
import { DetailAssetContentFrame } from "@/components/shared/layout/detail-asset-content-frame"
import {
  deferredInteractionUnmountDelayMs,
  useDeferredInteractionMount,
} from "@/hooks/use-deferred-interaction-mount"

import type { SubdomainsDetailViewState } from "./subdomains-detail-view-state"

const SUBDOMAINS_ROUTE_FALLBACK_PAGE_SIZE = 10

const BulkAddSubdomainsDialog = dynamic(
  () =>
    import("./bulk-add-subdomains-dialog").then((mod) => ({
      default: mod.BulkAddSubdomainsDialog,
    })),
  { ssr: false, loading: () => null }
)

export function SubdomainsDetailViewLoadingState({
  state,
  rowCount,
}: {
  state: SubdomainsDetailViewState
  rowCount: number
}) {
  return (
    <SubdomainsDataTable
      data={[]}
      columns={state.subdomainColumns}
      onSelectionChange={state.setSelectedSubdomains}
      selectedRows={[]}
      searchValue={state.searchQuery}
      onSearch={state.commitSearch}
      isSearching={state.isSearching}
      onExportAll={state.handleExportAll}
      onExportSelected={state.handleExportSelected}
      onBulkDelete={state.targetId ? () => state.setDeleteDialogOpen(true) : undefined}
      pagination={state.pagination}
      cursorPaginationSummary={state.cursorPaginationSummary}
      paginationNavigation={state.paginationNavigation}
      onPaginationChange={state.handlePaginationChange}
      sortingMode={state.sortingMode}
      sorting={state.sorting}
      onSortingChange={state.handleSortingChange}
      onBulkAdd={state.targetId ? () => state.setBulkAddOpen(true) : undefined}
      loading
      initialLoading
      loadingRowCount={rowCount}
    />
  )
}

export function SubdomainsDetailViewRouteFallback({ rowCount }: { rowCount: number }) {
  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const columns = createSubdomainColumns({
    formatDate: (value) => value,
    t: {
      columns: {
        subdomain: tColumns("subdomain.subdomain"),
        createdAt: tColumns("common.createdAt"),
      },
      actions: {
        selectAll: tCommon("actions.selectAll"),
        selectRow: tCommon("actions.selectRow"),
      },
    },
  })

  return (
    <DetailAssetContentFrame>
      <SubdomainsDataTable
        data={[]}
        columns={columns}
        onBulkAdd={() => {}}
        onSelectionChange={() => {}}
        selectedRows={[]}
        searchValue=""
        onSearch={() => {}}
        onExportAll={() => {}}
        onExportSelected={() => {}}
        pagination={{ pageIndex: 0, pageSize: SUBDOMAINS_ROUTE_FALLBACK_PAGE_SIZE }}
        paginationNavigation={{ mode: "cursor", canFirstPage: false, canPreviousPage: false, canNextPage: false }}
        cursorPaginationSummary={{ total: 0 }}
        onPaginationChange={() => {}}
        sortingMode="server"
        sorting={[]}
        onSortingChange={() => {}}
        loading
        initialLoading
        loadingRowCount={rowCount}
      />
    </DetailAssetContentFrame>
  )
}

export function SubdomainsDetailViewContent({ state }: { state: SubdomainsDetailViewState }) {
  return (
    <SubdomainsDataTable
      data={state.subdomains}
      columns={state.subdomainColumns}
      onSelectionChange={state.setSelectedSubdomains}
      selectedRows={state.selectedSubdomains}
      searchValue={state.searchQuery}
      onSearch={state.commitSearch}
      isSearching={state.isSearching}
      onExportAll={state.handleExportAll}
      onExportSelected={state.handleExportSelected}
      onBulkDelete={state.targetId ? () => state.setDeleteDialogOpen(true) : undefined}
      pagination={state.pagination}
      cursorPaginationSummary={state.cursorPaginationSummary}
      paginationNavigation={state.paginationNavigation}
      onPaginationChange={state.handlePaginationChange}
      sortingMode={state.sortingMode}
      sorting={state.sorting}
      onSortingChange={state.handleSortingChange}
      onBulkAdd={state.targetId ? () => state.setBulkAddOpen(true) : undefined}
    />
  )
}

export function SubdomainsDetailViewDialogs({ state }: { state: SubdomainsDetailViewState }) {
  const shouldMountBulkAddDialog = useDeferredInteractionMount(state.bulkAddOpen, {
    unmountDelayMs: deferredInteractionUnmountDelayMs,
  })

  return (
    <>
      {state.targetId && shouldMountBulkAddDialog ? (
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
