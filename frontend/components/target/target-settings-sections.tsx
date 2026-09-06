"use client"

import React from "react"
import dynamic from "next/dynamic"
import { useTranslations } from "next-intl"

import {
  AlertDialog,
  AlertDialogClose, AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { getDataTableSkeletonRowCount } from "@/components/shared/loading/data-table-skeleton"
import { getLoadingOwnerAttributes } from "@/components/shared/loading/loading-owner"
import { InteractionLoadingDialog } from "@/components/shared/loading/interaction-loading-dialog"
import { ScheduledScanDataTable } from "@/components/scan/scheduled/scheduled-scan-data-table"
import {
  createScheduledScanColumns,
  type ScheduledScanTranslations,
} from "@/components/scan/scheduled/scheduled-scan-columns"
import { BlacklistSettingsLoadingState } from "@/components/settings/blacklist/blacklist-settings-loading-state"
import {
  TARGET_SETTINGS_SHELL_CLASS,
  type TargetSettingsSection,
} from "./target-settings-layout"
import {
  deferredInteractionUnmountDelayMs,
  useDeferredInteractionMount,
} from "@/hooks/use-deferred-interaction-mount"

import type { TargetSettingsState } from "./target-settings-state"

const TARGET_SETTINGS_ROUTE_FALLBACK_PAGE_SIZE = 10

const CreateScheduledScanDialog = dynamic(
  () =>
    import("@/components/scan/scheduled/create-scheduled-scan-dialog").then(
      (mod) => mod.CreateScheduledScanDialog
    ),
  { ssr: false, loading: () => null }
)

const EditScheduledScanDialog = dynamic(
  () =>
    import("@/components/scan/scheduled/edit-scheduled-scan-dialog").then(
      (mod) => mod.EditScheduledScanDialog
    ),
  { ssr: false, loading: () => null }
)

export function TargetSettingsRouteFallback({
  rowCount,
  section = "blacklist",
}: {
  rowCount: number
  section?: TargetSettingsSection
}) {
  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const tScan = useTranslations("scan")
  const translations: ScheduledScanTranslations = {
    columns: {
      taskName: tColumns("scheduledScan.taskName"),
      cronExpression: tColumns("scheduledScan.cronExpression"),
      scope: tColumns("scheduledScan.scope"),
      status: tColumns("common.status"),
      nextRun: tColumns("scheduledScan.nextRun"),
      handoffResults: tColumns("scheduledScan.handoffResults"),
      trigger: tColumns("scheduledScan.trigger"),
      success: tColumns("scheduledScan.success"),
      failure: tColumns("scheduledScan.failure"),
      lastRun: tColumns("scheduledScan.lastRun"),
    },
    actions: {
      editTask: tScan("editTask"),
      delete: tCommon("actions.delete"),
      openMenu: tCommon("actions.openMenu"),
      selectAll: tCommon("actions.selectAll"),
      selectRow: tCommon("actions.selectRow"),
    },
    status: {
      enabled: tCommon("status.enabled"),
      disabled: tCommon("status.disabled"),
    },
    cron: {
      everyMinute: tScan("cron.everyMinute"),
      everyNMinutes: tScan.raw("cron.everyNMinutes") as string,
      everyHour: tScan.raw("cron.everyHour") as string,
      everyNHours: tScan.raw("cron.everyNHours") as string,
      everyDay: tScan.raw("cron.everyDay") as string,
      everyWeek: tScan.raw("cron.everyWeek") as string,
      everyMonth: tScan.raw("cron.everyMonth") as string,
      weekdays: tScan.raw("cron.weekdays") as string[],
    },
  }
  const columns = createScheduledScanColumns({
    formatDate: (value) => value,
    handleEdit: () => {},
    handleDelete: () => {},
    handleToggleStatus: () => {},
    t: translations,
  }).filter((column) => (column as { accessorKey?: string }).accessorKey !== "scanMode")

  if (section === "blacklist") {
    return <BlacklistSettingsLoadingState embedded scope="target" />
  }

  return (
    <div className={TARGET_SETTINGS_SHELL_CLASS}>
      <ScheduledScanDataTable
        data={[]}
        columns={columns}
        searchPlaceholder={tScan("scheduled.searchPlaceholder")}
        searchValue=""
        addButtonText={tScan("scheduled.createTitle")}
        page={1}
        pageSize={TARGET_SETTINGS_ROUTE_FALLBACK_PAGE_SIZE}
        total={0}
        totalPages={1}
        showQuickFilters={false}
        loading
        initialLoading
        loadingRowCount={rowCount}
      />
    </div>
  )
}

export function TargetSettingsLoadingState({
  state,
  rowCount,
}: {
  state: TargetSettingsState
  rowCount: number
}) {
  return (
    <div className={TARGET_SETTINGS_SHELL_CLASS}>
      <ScheduledScanDataTable
        data={[]}
        columns={state.columns}
        onAddNew={state.handleAddNew}
        searchPlaceholder={state.tScan("scheduled.searchPlaceholder")}
        searchValue={state.searchQuery}
        onSearch={state.commitSearch}
        isSearching={state.isSearching}
        addButtonText={state.tScan("scheduled.createTitle")}
        page={state.page}
        pageSize={state.pageSize}
        total={state.scheduledPaginationInfo.total}
        totalPages={state.scheduledPaginationInfo.totalPages}
        onPageChange={state.handlePageChange}
        onPageSizeChange={state.handlePageSizeChange}
        showQuickFilters={false}
        loading
        initialLoading
        loadingRowCount={rowCount}
      />
    </div>
  )
}

function TargetSettingsScheduledScansTableLoadingState({
  state,
}: {
  state: TargetSettingsState
}) {
  return (
    <div
      {...getLoadingOwnerAttributes({ owner: "target-settings-inline-table", layer: "section", intent: "data" })}
      data-slot="target-settings-scheduled-scans-table-loading-state"
      className="w-full"
    >
      <ScheduledScanDataTable
        data={[]}
        columns={state.columns}
        onAddNew={state.handleAddNew}
        searchPlaceholder={state.tScan("scheduled.searchPlaceholder")}
        searchValue={state.searchQuery}
        onSearch={state.commitSearch}
        isSearching={state.isSearching}
        addButtonText={state.tScan("scheduled.createTitle")}
        page={state.page}
        pageSize={state.pageSize}
        total={state.scheduledPaginationInfo.total}
        totalPages={state.scheduledPaginationInfo.totalPages}
        onPageChange={state.handlePageChange}
        onPageSizeChange={state.handlePageSizeChange}
        showQuickFilters={false}
        loading
        initialLoading
        loadingRowCount={getDataTableSkeletonRowCount(state.pageSize)}
      />
    </div>
  )
}

export function TargetSettingsContent({
  state,
}: {
  state: TargetSettingsState
}) {
  return (
    <div className={TARGET_SETTINGS_SHELL_CLASS}>
      {state.isLoadingScans ? (
        <TargetSettingsScheduledScansTableLoadingState state={state} />
      ) : (
        <ScheduledScanDataTable
          data={state.scheduledScans}
          columns={state.columns}
          onAddNew={state.handleAddNew}
          searchPlaceholder={state.tScan("scheduled.searchPlaceholder")}
          searchValue={state.searchQuery}
          onSearch={state.commitSearch}
          isSearching={state.isSearching}
          addButtonText={state.tScan("scheduled.createTitle")}
          page={state.page}
          pageSize={state.pageSize}
          total={state.scheduledPaginationInfo.total}
          totalPages={state.scheduledPaginationInfo.totalPages}
          onPageChange={state.handlePageChange}
          onPageSizeChange={state.handlePageSizeChange}
          showQuickFilters={false}
        />
      )}
    </div>
  )
}

export function TargetSettingsDialogs({
  state,
}: {
  state: TargetSettingsState
}) {
  const sidebarMountOptions = { unmountDelayMs: deferredInteractionUnmountDelayMs }
  const shouldMountCreateDialog = useDeferredInteractionMount(state.createDialogOpen, sidebarMountOptions)
  const shouldMountEditDialog = useDeferredInteractionMount(state.editDialogOpen, sidebarMountOptions)

  return (
    <>
      <InteractionLoadingDialog
        open={state.pendingInteraction === "edit-dialog"}
        onOpenChange={(open) => {
          if (!open) {
            state.cancelPendingInteraction("edit-dialog")
          }
        }}
        owner="target-settings-edit-dialog-loading"
        title={state.tScan("editTask")}
      />

      {shouldMountCreateDialog ? (
        <CreateScheduledScanDialog
          open={state.createDialogOpen}
          onOpenChange={state.setCreateDialogOpen}
          presetTargetId={state.targetId}
          presetTargetName={state.target?.name}
          onSuccess={() => state.refetch()}
        />
      ) : null}

      {shouldMountEditDialog ? (
        <EditScheduledScanDialog
          open={state.editDialogOpen}
          onOpenChange={state.setEditDialogOpen}
          scheduledScan={state.editingScheduledScan}
          onSuccess={() => state.refetch()}
        />
      ) : null}

      <AlertDialog open={state.deleteDialogOpen} onOpenChange={state.setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{state.tConfirm("deleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {state.tConfirm("deleteScheduledScanMessage", { name: state.deletingScheduledScan?.name ?? "" })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose variant="outline">{state.tCommon("actions.cancel")}</AlertDialogClose>
            <AlertDialogClose
              onClick={state.confirmDelete}
              className="bg-destructive hover:bg-destructive/90 text-destructive-foreground"
            >
              {state.tCommon("actions.delete")}
            </AlertDialogClose>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
