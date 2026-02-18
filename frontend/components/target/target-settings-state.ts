import React from "react"
import { useLocale, useTranslations } from "next-intl"

import { useTargetBlacklist, useUpdateTargetBlacklist, useTarget } from "@/hooks/use-targets"
import {
  useScheduledScans,
  useToggleScheduledScan,
  useDeleteScheduledScan,
} from "@/hooks/use-scheduled-scans"
import { useSearchState } from "@/hooks/_shared/use-search-state"
import { buildPaginationInfo } from "@/hooks/_shared/pagination"
import { createScheduledScanColumns } from "@/components/scan/scheduled/scheduled-scan-columns"

import type { ScheduledScan } from "@/types/scheduled-scan.types"

interface TargetSettingsStateOptions {
  targetId: number
}

export function useTargetSettingsState({ targetId }: TargetSettingsStateOptions) {
  const t = useTranslations("pages.targetDetail.settings")
  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const tScan = useTranslations("scan")
  const tConfirm = useTranslations("common.confirm")
  const locale = useLocale()

  const [blacklistText, setBlacklistText] = React.useState("")
  const [hasChanges, setHasChanges] = React.useState(false)

  const [createDialogOpen, setCreateDialogOpen] = React.useState(false)
  const [editDialogOpen, setEditDialogOpen] = React.useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [editingScheduledScan, setEditingScheduledScan] = React.useState<ScheduledScan | null>(null)
  const [deletingScheduledScan, setDeletingScheduledScan] = React.useState<ScheduledScan | null>(null)

  const [page, setPage] = React.useState(1)
  const [pageSize, setPageSize] = React.useState(10)
  const [searchQuery, setSearchQuery] = React.useState("")

  const { data: target } = useTarget(targetId)

  const { data, isLoading, error } = useTargetBlacklist(targetId)
  const updateBlacklist = useUpdateTargetBlacklist()

  const {
    data: scheduledScansData,
    isLoading: isLoadingScans,
    isFetching,
    refetch,
  } = useScheduledScans({
    targetId,
    page,
    pageSize,
    search: searchQuery || undefined,
  })

  const { isSearching, handleSearchChange } = useSearchState({
    isFetching,
    setSearchValue: setSearchQuery,
    onResetPage: () => setPage(1),
  })

  const { mutate: toggleScheduledScan } = useToggleScheduledScan()
  const { mutate: deleteScheduledScan } = useDeleteScheduledScan()

  const scheduledScans = scheduledScansData?.results || []
  const scheduledPaginationInfo = buildPaginationInfo({
    total: scheduledScansData?.total ?? 0,
    page,
    pageSize,
    totalPages: scheduledScansData?.totalPages,
    minTotalPages: 1,
  })

  const translations = React.useMemo(
    () => ({
      columns: {
        taskName: tColumns("scheduledScan.taskName"),
        scanEngine: tColumns("scheduledScan.scanEngine"),
        cronExpression: tColumns("scheduledScan.cronExpression"),
        scope: tColumns("scheduledScan.scope"),
        status: tColumns("common.status"),
        nextRun: tColumns("scheduledScan.nextRun"),
        runCount: tColumns("scheduledScan.runCount"),
        lastRun: tColumns("scheduledScan.lastRun"),
      },
      actions: {
        editTask: tScan("editTask"),
        delete: tCommon("actions.delete"),
        openMenu: tCommon("actions.openMenu"),
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

  React.useEffect(() => {
    if (data?.patterns) {
      setBlacklistText(data.patterns.join("\n"))
      setHasChanges(false)
    }
  }, [data])

  const handleTextChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setBlacklistText(e.target.value)
    setHasChanges(true)
  }

  const handleSave = () => {
    const patterns = blacklistText
      .split("\n")
      .map((line) => line.trim())
      .filter((line) => line.length > 0)

    updateBlacklist.mutate(
      { targetId, patterns },
      {
        onSuccess: () => {
          setHasChanges(false)
        },
      }
    )
  }

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
    setEditDialogOpen(true)
  }, [])

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
    setCreateDialogOpen(true)
  }, [])

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
    isLoading,
    error,
    isLoadingScans,
    refetch,
    updateBlacklist,
    blacklistText,
    hasChanges,
    handleTextChange,
    handleSave,
    scheduledScans,
    columns,
    searchQuery,
    handleSearchChange,
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
    deleteDialogOpen,
    setDeleteDialogOpen,
    editingScheduledScan,
    deletingScheduledScan,
    handleAddNew,
    confirmDelete,
  }
}

export type TargetSettingsState = ReturnType<typeof useTargetSettingsState>
