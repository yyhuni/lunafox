import React from "react"
import { useLocale, useTranslations } from "next-intl"
import { toast } from "sonner"

import { useTargetDirectories, useScanDirectories } from "@/hooks/use-directories"
import { useTarget } from "@/hooks/use-targets"
import { useSearchState } from "@/hooks/_shared/use-search-state"
import { buildPaginationInfo, normalizePagination } from "@/hooks/_shared/pagination"
import { DirectoryService } from "@/services/directory.service"
import { getDateLocale } from "@/lib/date-utils"
import { escapeCSV, formatDateForCSV } from "@/lib/csv-utils"
import { downloadBlob } from "@/lib/download-utils"
import { createDirectoryColumns } from "./directories-columns"

import type { Directory } from "@/types/directory.types"

interface DirectoriesViewStateOptions {
  targetId?: number
  scanId?: number
}

function formatDurationNsToMs(durationNs: number | null | undefined): string {
  if (durationNs === null || durationNs === undefined) return ""
  return String(Math.floor(durationNs / 1_000_000))
}

export function useDirectoriesViewState({ targetId, scanId }: DirectoriesViewStateOptions) {
  const [pagination, setPagination] = React.useState({
    pageIndex: 0,
    pageSize: 10,
  })
  const [selectedDirectories, setSelectedDirectories] = React.useState<Directory[]>([])
  const [bulkAddDialogOpen, setBulkAddDialogOpen] = React.useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [isDeleting, setIsDeleting] = React.useState(false)
  const [filterQuery, setFilterQuery] = React.useState("")

  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const tToast = useTranslations("toast")
  const tStatus = useTranslations("common.status")
  const locale = useLocale()

  const translations = React.useMemo(
    () => ({
      columns: {
        url: tColumns("common.url"),
        status: tColumns("common.status"),
        length: tColumns("directory.length"),
        words: tColumns("directory.words"),
        lines: tColumns("directory.lines"),
        contentType: tColumns("endpoint.contentType"),
        duration: tColumns("directory.duration"),
        createdAt: tColumns("common.createdAt"),
      },
      actions: {
        selectAll: tCommon("actions.selectAll"),
        selectRow: tCommon("actions.selectRow"),
      },
    }),
    [tColumns, tCommon]
  )

  const { data: target } = useTarget(targetId || 0, { enabled: !!targetId })

  const targetQuery = useTargetDirectories(
    targetId || 0,
    {
      page: pagination.pageIndex + 1,
      pageSize: pagination.pageSize,
      filter: filterQuery || undefined,
    },
    { enabled: !!targetId }
  )

  const scanQuery = useScanDirectories(
    scanId || 0,
    {
      page: pagination.pageIndex + 1,
      pageSize: pagination.pageSize,
      filter: filterQuery || undefined,
    },
    { enabled: !!scanId }
  )

  const activeQuery = targetId ? targetQuery : scanQuery
  const { data, isLoading, isFetching, error, refetch } = activeQuery

  const { isSearching, handleSearchChange: handleFilterChange } = useSearchState({
    isFetching,
    setSearchValue: setFilterQuery,
    onResetPage: () => setPagination((prev) => ({ ...prev, pageIndex: 0 })),
  })

  const formatDate = React.useCallback(
    (dateString: string) => {
      return new Date(dateString).toLocaleString(getDateLocale(locale), {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
        hour12: false,
      })
    },
    [locale]
  )

  const columns = React.useMemo(
    () =>
      createDirectoryColumns({
        formatDate,
        t: translations,
      }),
    [formatDate, translations]
  )

  const directories: Directory[] = React.useMemo(() => {
    return data?.results ?? []
  }, [data])

  const paginationInfo = data
    ? buildPaginationInfo({
        ...normalizePagination(data, pagination.pageIndex + 1, pagination.pageSize),
        minTotalPages: 1,
      })
    : undefined

  const handleSelectionChange = React.useCallback((selectedRows: Directory[]) => {
    setSelectedDirectories(selectedRows)
  }, [])

  const generateCSV = React.useCallback((items: Directory[]): string => {
    const bom = "\ufeff"
    const headers = [
      "url",
      "status",
      "content_length",
      "words",
      "lines",
      "content_type",
      "duration",
      "created_at",
    ]

    const rows = items.map((item) =>
      [
        escapeCSV(item.url),
        escapeCSV(item.status),
        escapeCSV(item.contentLength),
        escapeCSV(item.words),
        escapeCSV(item.lines),
        escapeCSV(item.contentType),
        escapeCSV(formatDurationNsToMs(item.duration)),
        escapeCSV(formatDateForCSV(item.createdAt)),
      ].join(",")
    )

    return bom + [headers.join(","), ...rows].join("\n")
  }, [])

  const handleDownloadAll = React.useCallback(async () => {
    try {
      let blob: Blob | null = null

      if (scanId) {
        blob = await DirectoryService.exportDirectoriesByScanId(scanId)
      } else if (targetId) {
        blob = await DirectoryService.exportDirectoriesByTargetId(targetId)
      } else if (directories.length > 0) {
        const csvContent = generateCSV(directories)
        blob = new Blob([csvContent], { type: "text/csv;charset=utf-8" })
      }

      if (!blob) return

      const prefix = scanId ? `scan-${scanId}` : targetId ? `target-${targetId}` : "directories"
      downloadBlob(blob, `${prefix}-directories-${Date.now()}.csv`)
    } catch {
      toast.error(tToast("downloadFailed"))
    }
  }, [directories, generateCSV, scanId, targetId, tToast])

  const handleDownloadSelected = React.useCallback(() => {
    if (selectedDirectories.length === 0) {
      return
    }
    const csvContent = generateCSV(selectedDirectories)
    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8" })
    const prefix = scanId ? `scan-${scanId}` : targetId ? `target-${targetId}` : "directories"
    downloadBlob(blob, `${prefix}-directories-selected-${Date.now()}.csv`)
  }, [generateCSV, scanId, selectedDirectories, targetId])

  const handleBulkDelete = React.useCallback(async () => {
    if (selectedDirectories.length === 0) return

    setIsDeleting(true)
    try {
      const ids = selectedDirectories.map((directory) => directory.id)
      const result = await DirectoryService.bulkDelete(ids)
      toast.success(tToast("deleteSuccess", { count: result.deletedCount }))
      setSelectedDirectories([])
      setDeleteDialogOpen(false)
      refetch()
    } catch {
      toast.error(tToast("deleteFailed"))
    } finally {
      setIsDeleting(false)
    }
  }, [selectedDirectories, tToast, refetch])

  return {
    tCommon,
    tStatus,
    targetId,
    target,
    data,
    error,
    isLoading,
    isSearching,
    refetch,
    columns,
    directories,
    filterQuery,
    handleFilterChange,
    pagination,
    setPagination,
    paginationInfo,
    handleSelectionChange,
    handleDownloadAll,
    handleDownloadSelected,
    handleBulkDelete,
    bulkAddDialogOpen,
    setBulkAddDialogOpen,
    deleteDialogOpen,
    setDeleteDialogOpen,
    selectedDirectories,
    isDeleting,
  }
}

export type DirectoriesViewState = ReturnType<typeof useDirectoriesViewState>
