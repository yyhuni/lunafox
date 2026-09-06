import React from "react"
import { useLocale, useTranslations } from "next-intl"

import { useTarget } from "@/hooks/use-targets"
import {
  useScheduledScans,
  useToggleScheduledScan,
  useDeleteScheduledScan,
} from "@/hooks/use-scheduled-scans"
import { useSearchState } from "@/hooks/_shared/use-search-state"
import { buildPaginationInfo } from "@/hooks/_shared/pagination"
import { createScheduledScanColumns } from "@/components/scan/scheduled/scheduled-scan-columns"
import { useInteractionOpenLoader } from "@/components/shared/loading/use-interaction-open-loader"

import type { ScheduledScan } from "@/types/scheduled-scan.types"

interface TargetSettingsStateOptions {
  targetId: number
}

const loadCreateScheduledScanDialog = () =>
  import("@/components/scan/scheduled/create-scheduled-scan-dialog").then(
    (mod) => mod.CreateScheduledScanDialog
  )

const loadEditScheduledScanDialog = () =>
  import("@/components/scan/scheduled/edit-scheduled-scan-dialog").then(
    (mod) => mod.EditScheduledScanDialog
  )

export function useTargetSettingsState({ targetId }: TargetSettingsStateOptions) {
  const {
    pendingInteraction,
    cancelPending,
    openAfterLoad,
  } = useInteractionOpenLoader<"create-dialog" | "edit-dialog">()
  const t = useTranslations("pages.targetDetail.settings")
  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const tScan = useTranslations("scan")
  const tConfirm = useTranslations("common.confirm")
  const locale = useLocale()

  const [createDialogOpen, setCreateDialogOpen] = React.useState(false)
  const [editDialogOpen, setEditDialogOpen] = React.useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [editingScheduledScan, setEditingScheduledScan] = React.useState<ScheduledScan | null>(null)
  const [deletingScheduledScan, setDeletingScheduledScan] = React.useState<ScheduledScan | null>(null)

  const [page, setPage] = React.useState(1)
  const [pageSize, setPageSize] = React.useState(10)
  const [searchQuery, setSearchQuery] = React.useState("")

  const { data: target } = useTarget(targetId)

  const {
    data: scheduledScansData,
    isLoading: isLoadingScans,
    isFetching,
    error,
    refetch,
  } = useScheduledScans({
    targetId,
    page,
    pageSize,
    search: searchQuery || undefined,
  })

  const { isSearching, commitSearch } = useSearchState({
    isFetching,
    searchValue: searchQuery,
    setSearchValue: setSearchQuery,
    onResetPage: () => setPage(1),
  })

  const { mutate: toggleScheduledScan } = useToggleScheduledScan()
  const { mutate: deleteScheduledScan } = useDeleteScheduledScan()

  const scheduledScans = scheduledScansData?.scheduledScans || []
  const scheduledPaginationInfo = buildPaginationInfo({
    total: scheduledScansData?.totalSize ?? 0,
    page,
    pageSize,
    minTotalPages: 1,
  })

  const translations = React.useMemo(
    () => ({
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
    }),
    [tColumns, tCommon, tScan]
  )

  const formatDate = React.useCallback(
    (dateString: string) => {
      const date = new Date(dateString)
      return date.toLocaleString(locale === "zh" ? "zh-CN" : "en-US", {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
      })
    },
    [locale]
  )

  const handleEdit = React.useCallback((scan: ScheduledScan) => {
    setEditingScheduledScan(scan)
    void openAfterLoad("edit-dialog", loadEditScheduledScanDialog, () => {
      setEditDialogOpen(true)
    })
  }, [openAfterLoad])

  const handleDelete = React.useCallback((scan: ScheduledScan) => {
    setDeletingScheduledScan(scan)
    setDeleteDialogOpen(true)
  }, [])

  const confirmDelete = React.useCallback(() => {
    if (deletingScheduledScan) {
      deleteScheduledScan(deletingScheduledScan.id)
      setDeleteDialogOpen(false)
      setDeletingScheduledScan(null)
    }
  }, [deletingScheduledScan, deleteScheduledScan])

  const handleToggleStatus = React.useCallback(
    (scan: ScheduledScan, enabled: boolean) => {
      toggleScheduledScan({ id: scan.id, isEnabled: enabled })
    },
    [toggleScheduledScan]
  )

  const handlePageChange = React.useCallback((newPage: number) => {
    setPage(newPage)
  }, [])

  const handlePageSizeChange = React.useCallback((newPageSize: number) => {
    setPageSize(newPageSize)
    setPage(1)
  }, [])

  const handleAddNew = React.useCallback(() => {
    void openAfterLoad("create-dialog", loadCreateScheduledScanDialog, () => {
      setCreateDialogOpen(true)
    })
  }, [openAfterLoad])

  const columns = React.useMemo(() => {
    const allColumns = createScheduledScanColumns({
      formatDate,
      handleEdit,
      handleDelete,
      handleToggleStatus,
      t: translations,
    })
    return allColumns.filter((col) => (col as { accessorKey?: string }).accessorKey !== "scanMode")
  }, [formatDate, handleEdit, handleDelete, handleToggleStatus, translations])

  return {
    t,
    tCommon,
    tScan,
    tConfirm,
    targetId,
    target,
    isLoading: isLoadingScans,
    error,
    isLoadingScans,
    refetch,
    scheduledScans,
    columns,
    searchQuery,
    commitSearch,
    isSearching,
    page,
    pageSize,
    scheduledPaginationInfo,
    handlePageChange,
    handlePageSizeChange,
    createDialogOpen,
    setCreateDialogOpen,
    editDialogOpen,
    setEditDialogOpen,
    pendingInteraction,
    cancelPendingInteraction: cancelPending,
    deleteDialogOpen,
    setDeleteDialogOpen,
    editingScheduledScan,
    deletingScheduledScan,
    handleAddNew,
    confirmDelete,
  }
}

export type TargetSettingsState = ReturnType<typeof useTargetSettingsState>
