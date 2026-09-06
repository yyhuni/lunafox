"use client"

import dynamic from "next/dynamic"

import { ConfirmDialog } from "@/components/shared/feedback/confirm-dialog"
import {
  deferredInteractionUnmountDelayMs,
  useDeferredInteractionMount,
} from "@/hooks/use-deferred-interaction-mount"

import { OrganizationDataTable } from "./organization-data-table"

import type { OrganizationListState } from "./organization-list-state"

const loadOrganizationDetailDrawer = () =>
  import("./organization-detail-view").then((mod) => ({
    default: mod.OrganizationDetailDrawerView,
  }))

const OrganizationDetailView = dynamic(
  loadOrganizationDetailDrawer,
  { ssr: false, loading: () => null }
)

const loadAddOrganizationDialog = () =>
  import("./add-organization-dialog").then((mod) => ({
    default: mod.AddOrganizationDialog,
  }))

const AddOrganizationDialog = dynamic(
  loadAddOrganizationDialog,
  { ssr: false, loading: () => null }
)

function preloadOrganizationDetailDrawer() {
  void loadOrganizationDetailDrawer()
}

function preloadAddOrganizationDialog() {
  void loadAddOrganizationDialog()
}

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

export function OrganizationListLoadingState({
  state,
  rowCount,
}: {
  state: OrganizationListState
  rowCount: number
}) {
  return (
    <OrganizationDataTable
      data={[]}
      columns={state.columns}
      onAddNew={() => state.setAddDialogOpen(true)}
      onAddIntent={preloadAddOrganizationDialog}
      onBulkDelete={state.handleBulkDelete}
      onBulkInitiateScan={state.handleBulkInitiateScan}
      onViewDetail={state.handleViewDetail}
      onDetailIntent={preloadOrganizationDetailDrawer}
      onSelectionChange={state.setSelectedOrganizations}
      selectedRows={[]}
      searchPlaceholder={state.tOrg("name")}
      searchValue={state.searchQuery}
      onSearch={state.commitSearch}
      isSearching={state.isSearching}
      pagination={state.pagination}
      cursorPaginationSummary={state.cursorPaginationSummary}
      paginationNavigation={state.paginationNavigation}
      onPaginationChange={state.handlePaginationChange}
      sorting={state.sorting}
      onSortingChange={state.handleSortingChange}
      loading
      loadingRowCount={rowCount}
      stableSurfaceRowCount={rowCount}
    />
  )
}

export function OrganizationListTable({
  state,
  stableSurfaceRowCount,
}: {
  state: OrganizationListState
  stableSurfaceRowCount: number
}) {
  return (
    <OrganizationDataTable
      data={state.organizations}
      columns={state.columns}
      onAddNew={() => state.setAddDialogOpen(true)}
      onAddIntent={preloadAddOrganizationDialog}
      onBulkDelete={state.handleBulkDelete}
      onBulkInitiateScan={state.handleBulkInitiateScan}
      onViewDetail={state.handleViewDetail}
      onDetailIntent={preloadOrganizationDetailDrawer}
      onSelectionChange={state.setSelectedOrganizations}
      selectedRows={state.selectedOrganizations}
      searchPlaceholder={state.tOrg("name")}
      searchValue={state.searchQuery}
      onSearch={state.commitSearch}
      isSearching={state.isSearching}
      pagination={state.pagination}
      cursorPaginationSummary={state.cursorPaginationSummary}
      paginationNavigation={state.paginationNavigation}
      onPaginationChange={state.handlePaginationChange}
      sorting={state.sorting}
      onSortingChange={state.handleSortingChange}
      stableSurfaceRowCount={stableSurfaceRowCount}
    />
  )
}

export function OrganizationListDialogs({
  state,
}: {
  state: OrganizationListState
}) {
  const sidebarMountOptions = { unmountDelayMs: deferredInteractionUnmountDelayMs }
  const shouldMountDetailDrawer = useDeferredInteractionMount(Boolean(state.organizationToView), sidebarMountOptions)
  const shouldMountAddDialog = useDeferredInteractionMount(state.addDialogOpen, sidebarMountOptions)
  const shouldMountBulkScanDrawer = useDeferredInteractionMount(state.bulkInitiateScanDialogOpen, sidebarMountOptions)
  const shouldMountScanDrawer = useDeferredInteractionMount(state.initiateScanDialogOpen, sidebarMountOptions)
  const shouldMountScheduleSheet = useDeferredInteractionMount(state.scheduleScanDialogOpen, sidebarMountOptions)

  return (
    <>
      <ConfirmDialog
        open={state.deleteDialogOpen}
        onOpenChange={state.setDeleteDialogOpen}
        title={state.tConfirm("deleteTitle")}
        description={state.tConfirm("deleteOrgMessage", { name: state.organizationToDelete?.name ?? "" })}
        onConfirm={state.confirmDelete}
        loading={state.deleteOrganization.isPending}
        variant="destructive"
        confirmText={state.tConfirm("confirmDelete")}
        cancelText={state.tCommon("actions.cancel")}
        processingText={state.tConfirm("deleting")}
      />

      {shouldMountDetailDrawer ? <OrganizationDetailDrawer state={state} /> : null}

      <ConfirmDialog
        open={state.bulkDeleteDialogOpen}
        onOpenChange={state.setBulkDeleteDialogOpen}
        title={state.tConfirm("bulkDeleteTitle")}
        description={state.tConfirm("bulkDeleteOrgMessage", { count: state.selectedOrganizations.length })}
        onConfirm={state.confirmBulkDelete}
        loading={state.batchDeleteOrganizations.isPending}
        variant="destructive"
        confirmText={state.tConfirm("confirmDelete")}
        cancelText={state.tCommon("actions.cancel")}
        processingText={state.tConfirm("deleting")}
      />

      {shouldMountAddDialog ? (
        <AddOrganizationDialog
          open={state.addDialogOpen}
          onOpenChange={state.setAddDialogOpen}
          onAdd={() => {
            state.setAddDialogOpen(false)
          }}
        />
      ) : null}

      {shouldMountBulkScanDrawer ? (
        <BulkInitiateScanDrawer
          organizationIds={state.selectedOrganizations.map((organization) => organization.id)}
          scopeLabel={state.tScanInitiate("bulkOrganizationsDesc", { count: state.selectedOrganizations.length })}
          open={state.bulkInitiateScanDialogOpen}
          onOpenChange={state.setBulkInitiateScanDialogOpen}
          onSuccess={state.handleBulkInitiateScanSuccess}
        />
      ) : null}

      {shouldMountScanDrawer ? (
        <InitiateScanDrawer
          organization={state.organizationToScan}
          organizationId={state.organizationToScan?.id}
          open={state.initiateScanDialogOpen}
          onOpenChange={state.setInitiateScanDialogOpen}
          onSuccess={() => {
            state.setOrganizationToScan(null)
          }}
        />
      ) : null}

      {shouldMountScheduleSheet ? (
        <CreateScheduledScanSheet
          open={state.scheduleScanDialogOpen}
          onOpenChange={state.setScheduleScanDialogOpen}
          presetOrganizationId={state.organizationToSchedule?.id}
          presetOrganizationName={state.organizationToSchedule?.name}
          onSuccess={() => {
            state.setOrganizationToSchedule(null)
          }}
        />
      ) : null}
    </>
  )
}

function OrganizationDetailDrawer({
  state,
}: {
  state: OrganizationListState
}) {
  const organization = state.organizationToView

  return (
    <OrganizationDetailView
      organizationId={String(organization?.id ?? "")}
      open={Boolean(organization)}
      onOpenChange={state.handleDetailOpenChange}
      title={state.translations.tooltips.organizationDetails}
      description={organization?.name}
      previewDescription={organization?.description}
    />
  )
}
