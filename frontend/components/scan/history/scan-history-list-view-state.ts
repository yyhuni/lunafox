import React from "react"
import { useLocale, useTranslations } from "next-intl"

import { createScanHistoryColumns } from "./scan-history-columns"
import { getDateLocale } from "@/lib/date-utils"
import { useScans } from "@/hooks/use-scans"
import { useSearchState } from "@/hooks/_shared/use-search-state"
import { buildPaginationInfo, normalizePagination } from "@/hooks/_shared/pagination"
import { useScanHistoryActions } from "@/components/scan/history/scan-history-list-state"

import type { ScanStatus, ScanRecord } from "@/types/scan.types"
import type { ColumnDef } from "@tanstack/react-table"

interface ScanHistoryListViewStateOptions {
  hideToolbar?: boolean
  targetId?: number
  pageSize?: number
  hideTargetColumn?: boolean
  pageSizeOptions?: number[]
  hidePagination?: boolean
}

export function useScanHistoryListViewState({
  hideToolbar = false,
  targetId,
  pageSize: customPageSize,
  hideTargetColumn = false,
  pageSizeOptions,
  hidePagination = false,
}: ScanHistoryListViewStateOptions) {
  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const tTooltips = useTranslations("tooltips")
  const tScan = useTranslations("scan")
  const tToast = useTranslations("toast")
  const tConfirm = useTranslations("common.confirm")
  const locale = useLocale()

  const translations = React.useMemo(
    () => ({
      columns: {
        target: tColumns("scanHistory.target"),
        summary: tColumns("scanHistory.summary"),
        workflowName: tColumns("scanHistory.workflowName"),
        workerName: tColumns("scanHistory.workerName"),
        createdAt: tColumns("common.createdAt"),
        status: tColumns("common.status"),
        progress: tColumns("scanHistory.progress"),
      },
      actions: {
        snapshot: tCommon("actions.snapshot"),
        stop: tCommon("actions.stop"),
        stopScanPending: tScan("stopScanPending"),
        delete: tCommon("actions.delete"),
        selectAll: tCommon("actions.selectAll"),
        selectRow: tCommon("actions.selectRow"),
      },
      tooltips: {
        targetDetails: tTooltips("targetDetails"),
        viewProgress: tTooltips("viewProgress"),
      },
      status: {
        cancelled: tCommon("status.cancelled"),
        completed: tCommon("status.completed"),
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
    }),
    [tColumns, tCommon, tTooltips, tScan]
  )

  const [pagination, setPagination] = React.useState({
    pageIndex: 0,
    pageSize: customPageSize || 10,
  })

  const [searchQuery, setSearchQuery] = React.useState("")
  const [statusFilter, setStatusFilter] = React.useState<ScanStatus | "all">("all")

  const handleStatusFilterChange = (status: ScanStatus | "all") => {
    setStatusFilter(status)
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }

  const { data, isLoading, isFetching, error, refetch } = useScans({
    page: pagination.pageIndex + 1,
    pageSize: pagination.pageSize,
    search: searchQuery || undefined,
    target: targetId,
    status: statusFilter === "all" ? undefined : statusFilter,
  })

  const { isSearching, handleSearchChange } = useSearchState({
    isFetching,
    setSearchValue: setSearchQuery,
    onResetPage: () => setPagination((prev) => ({ ...prev, pageIndex: 0 })),
  })

  const paginationInfo = data
    ? buildPaginationInfo({
        ...normalizePagination(data, pagination.pageIndex + 1, pagination.pageSize),
        minTotalPages: 1,
      })
    : undefined

  const scans = data?.results || []

  const actions = useScanHistoryActions({ tToast })

  const formatDate = React.useCallback(
    (dateString: string): string => {
      return new Date(dateString).toLocaleString(getDateLocale(locale), {
        year: "numeric",
        month: "numeric",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
        hour12: false,
      })
    },
    [locale]
  )

  const handlePaginationChange = (newPagination: { pageIndex: number; pageSize: number }) => {
    setPagination(newPagination)
  }

  const scanColumns = React.useMemo(
    () =>
      createScanHistoryColumns({
        formatDate,
        handleDelete: actions.handleDeleteScan,
        handleStop: actions.handleStopScan,
        handleViewProgress: actions.handleViewProgress,
        statusClickable: false,
        t: translations,
        hideTargetColumn,
      }),
    [formatDate, translations, hideTargetColumn, actions]
  )

  return {
    tCommon,
    tConfirm,
    tScan,
    hideToolbar,
    pageSizeOptions,
    hidePagination,
    isLoading,
    error,
    refetch,
    scans,
    scanColumns: scanColumns as ColumnDef<ScanRecord>[],
    searchQuery,
    isSearching,
    handleSearchChange,
    pagination,
    setPagination,
    paginationInfo,
    handlePaginationChange,
    statusFilter,
    handleStatusFilterChange,
    ...actions,
  }
}

export type ScanHistoryListViewState = ReturnType<typeof useScanHistoryListViewState>
