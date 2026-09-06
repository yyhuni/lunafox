import React from "react"
import { useLocale, useTranslations } from "next-intl"
import type { ColumnDef, SortingState } from "@tanstack/react-table"

import { semanticIcons } from "@/components/icons"
import { createScanHistoryColumns } from "./scan-history-columns"
import { getDateLocale } from "@/lib/date-utils"
import { useScans } from "@/hooks/use-scans"
import { useScanExecutedEngineDisplay } from "@/hooks/use-scan-executed-engine-display"
import { useScanHistoryActions } from "@/components/scan/history/scan-history-list-state"
import {
  applyBusinessListControlChange,
  compileBusinessListFilter,
  compileBusinessListOrderBy,
  createBusinessListQuery,
  getCurrentCursorNextPageToken,
  getCursorPaginationNavigation,
  getCursorPageTransition,
  setBusinessListPage,
  toggleBusinessListSorting,
  type BusinessListFilterCompilerConfig,
  type BusinessListSortableFieldConfig,
  type BusinessListSorting,
} from "@/components/shared/data-table/business-list-query"
import { useCursorPaginationScopeChange } from "@/components/shared/data-table/use-cursor-pagination-scope"

import type { ScanStatus, ScanRecord } from "@/types/scan.types"
import type {
  CursorPaginationNavigation,
  CursorPaginationSummary,
} from "@/types/data-table.types"
import type { SelectedRowActionBarAction } from "@/components/shared/data-table"

interface ScanHistoryListViewStateOptions {
  hideToolbar?: boolean
  targetId?: number
  pageSize?: number
  hideTargetColumn?: boolean
  pageSizeOptions?: number[]
  hidePagination?: boolean
}

export type ScanStatusFilter = ScanStatus[]

const SCAN_HISTORY_FILTER_FIELDS: BusinessListFilterCompilerConfig = {
  search: { field: "targetName", operator: "=" },
  facets: {
    status: { field: "status", operator: "==" },
  },
}

const SCAN_HISTORY_SORTABLE_FIELDS: Record<string, BusinessListSortableFieldConfig> = {
  createdAt: { orderBy: "createdAt", firstDirection: "desc" },
}

const SCAN_HISTORY_DEFAULT_SORTING: BusinessListSorting = { field: "createdAt", direction: "desc" }

function scanHistoryQueryFieldToColumnId(field: string) {
  return field
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
  const tConfirm = useTranslations("common.confirm")
  const locale = useLocale()

  const translations = React.useMemo(
    () => ({
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
        batchStopDisabled: tScan("history.batchStop.disabled"),
        batchStopInProgress: tScan("history.batchStop.inProgress"),
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
    }),
    [tColumns, tCommon, tTooltips, tScan]
  )

  const [query, setQuery] = React.useState(() => createBusinessListQuery({
    pageSize: customPageSize || 10,
    sorting: SCAN_HISTORY_DEFAULT_SORTING,
  }))
  const [pageTokens, setPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })
  const cursorScopeKey = `target:${targetId ?? ""}`
  const hasCursorScopeChanged = useCursorPaginationScopeChange(cursorScopeKey)

  const [rowSelection, setRowSelection] = React.useState<Record<string, boolean>>({})

  const page = query.pageIndex ?? 1
  const pageSize = query.pageSize
  const activePage = hasCursorScopeChanged ? 1 : page
  const pageToken = hasCursorScopeChanged ? undefined : query.pageToken
  const searchQuery = query.search ?? ""
  const statusFilter = query.filters.status ?? []
  const pagination = React.useMemo(
    () => ({ pageIndex: activePage - 1, pageSize }),
    [activePage, pageSize]
  )
  const sorting = React.useMemo<SortingState>(() => {
    if (!query.sorting) return []
    return [{
      id: scanHistoryQueryFieldToColumnId(query.sorting.field),
      desc: query.sorting.direction === "desc",
    }]
  }, [query.sorting])

  const filterParam = compileBusinessListFilter(
    { search: query.search, filters: query.filters },
    SCAN_HISTORY_FILTER_FIELDS
  )
  const orderByParam = compileBusinessListOrderBy(query.sorting, SCAN_HISTORY_SORTABLE_FIELDS)

  const { data, isLoading, isFetching, isPlaceholderData, error, refetch } = useScans({
    pageSize,
    pageToken,
    filter: filterParam,
    orderBy: orderByParam,
    target: targetId,
  })

  const nextPageToken = getCurrentCursorNextPageToken(
    data?.nextPageToken,
    isPlaceholderData || hasCursorScopeChanged,
  )

  React.useEffect(() => {
    if (nextPageToken) {
      setPageTokens((tokens) => ({ ...tokens, [activePage + 1]: nextPageToken }))
    }
  }, [activePage, nextPageToken])

  const resetPaging = React.useCallback(() => {
    setPageTokens({ 1: undefined })
  }, [])

  React.useEffect(() => {
    if (!hasCursorScopeChanged) return

    setRowSelection({})
    resetPaging()
    setQuery((current) => setBusinessListPage(current, { pageIndex: 1, pageToken: undefined }))
  }, [hasCursorScopeChanged, resetPaging])

  const isSearching = isFetching

  const commitSearch = React.useCallback((value: string) => {
    const normalizedSearch = value.trim()
    if ((query.search ?? "") === normalizedSearch) return
    resetPaging()
    setQuery((current) => applyBusinessListControlChange(current, {
      search: normalizedSearch || undefined,
    }))
  }, [query.search, resetPaging])

  const handleStatusFilterChange = React.useCallback((status: ScanStatusFilter) => {
    resetPaging()
    setQuery((current) => applyBusinessListControlChange(current, {
      filters: {
        ...current.filters,
        status,
      },
    }))
  }, [resetPaging])

  const cursorPaginationSummary: CursorPaginationSummary = {
    total: data?.totalSize ?? data?.total ?? 0,
  }
  const paginationNavigation: CursorPaginationNavigation = getCursorPaginationNavigation({
    currentPage: activePage,
    pageTokens: hasCursorScopeChanged ? { 1: undefined } : pageTokens,
    nextPageToken,
  })

  const scans = React.useMemo(() => data?.results ?? [], [data?.results])
  const executedEngineDisplay = useScanExecutedEngineDisplay(scans)

  const selectedScans = React.useMemo(
    () => scans.filter((scan) => rowSelection[String(scan.id)]),
    [rowSelection, scans]
  )

  React.useEffect(() => {
    const scanIds = new Set(scans.map((scan) => String(scan.id)))
    setRowSelection((current) => {
      const next = Object.fromEntries(
        Object.entries(current).filter(([id]) => scanIds.has(id))
      )
      return Object.keys(next).length === Object.keys(current).length ? current : next
    })
  }, [scans])

  const handleSelectionClear = React.useCallback(() => {
    setRowSelection({})
  }, [])

  const {
    selectedScans: actionSelectedScans,
    deleteDialogOpen,
    setDeleteDialogOpen,
    scanToDelete,
    bulkDeleteDialogOpen,
    setBulkDeleteDialogOpen,
    stopDialogOpen,
    setStopDialogOpen,
    scanToStop,
    progressDialogOpen,
    setProgressDialogOpen,
    progressData,
    runtimeDetailOpen,
    setRuntimeDetailOpen,
    scanForRuntimeDetail,
    handleDeleteScan,
    confirmDelete,
    handleBulkDelete,
    confirmBulkDelete,
    handleStopScan,
    confirmStop,
    batchStopDialogOpen,
    setBatchStopDialogOpen,
    activeSelectedScans,
    terminalSelectedCount,
    canBatchStop,
    batchStopMutationPending,
    handleBatchStop,
    confirmBatchStop,
    handleViewProgress,
    handleViewRuntimeDetail,
  } = useScanHistoryActions({
    selectedScans,
    onSelectionClear: handleSelectionClear,
  })

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

  const handlePaginationChange = React.useCallback(
    (newPagination: { pageIndex: number; pageSize: number }) => {
      const nextPage = newPagination.pageIndex + 1
      if (newPagination.pageSize !== pageSize) {
        resetPaging()
        setQuery((current) => applyBusinessListControlChange(current, { pageSize: newPagination.pageSize }))
        return
      }

      if (nextPage === 1) {
        resetPaging()
        setQuery((current) => setBusinessListPage(current, { pageIndex: 1, pageToken: undefined }))
        return
      }

      const transition = getCursorPageTransition({
        currentPage: activePage,
        pageTokens,
        nextPageToken,
        requestedPage: nextPage,
      })
      if (!transition.reachable) return

      setQuery((current) => setBusinessListPage(current, {
        pageIndex: nextPage,
        pageToken: transition.pageToken,
      }))
    },
    [activePage, nextPageToken, pageSize, pageTokens, resetPaging]
  )

  const handleSortingChange = React.useCallback((nextSorting: SortingState) => {
    const nextColumnId = nextSorting[0]?.id
    resetPaging()
    if (!nextColumnId) {
      setQuery((current) => applyBusinessListControlChange(current, { sorting: SCAN_HISTORY_DEFAULT_SORTING }))
      return
    }

    setQuery((current) => toggleBusinessListSorting(
      current,
      nextColumnId,
      SCAN_HISTORY_SORTABLE_FIELDS,
      SCAN_HISTORY_DEFAULT_SORTING
    ))
  }, [resetPaging])

  const scanColumns = React.useMemo(
    () =>
      createScanHistoryColumns({
        formatDate,
        handleDelete: handleDeleteScan,
        handleStop: handleStopScan,
        t: translations,
        hideTargetColumn,
        executedEngineNamesByScanId: executedEngineDisplay.engineNamesByScanId,
        executedEngineDescriptionsByScanId: executedEngineDisplay.engineDescriptionsByScanId,
      }),
    [formatDate, translations, hideTargetColumn, handleDeleteScan, handleStopScan, executedEngineDisplay.engineDescriptionsByScanId, executedEngineDisplay.engineNamesByScanId]
  )

  const selectedRowActions = React.useMemo<SelectedRowActionBarAction[]>(
    () => [{
      key: "stop",
      label: translations.actions.stop,
      icon: semanticIcons.action.stop,
      tone: "muted",
      group: "lifecycle",
      disabled: !canBatchStop || batchStopMutationPending,
      disabledReason: batchStopMutationPending
        ? translations.actions.batchStopInProgress
        : translations.actions.batchStopDisabled,
      onClick: handleBatchStop,
    }],
    [batchStopMutationPending, canBatchStop, handleBatchStop, translations.actions]
  )

  return {
    tCommon,
    tConfirm,
    tScan,
    hideToolbar,
    pageSizeOptions,
    hidePagination,
    isLoading: isLoading || executedEngineDisplay.isLoading,
    error: error ?? executedEngineDisplay.error,
    refetch,
    scans,
    scanColumns: scanColumns as ColumnDef<ScanRecord>[],
    searchQuery,
    isSearching,
    commitSearch,
    pagination,
    cursorPaginationSummary,
    paginationNavigation,
    handlePaginationChange,
    sortingMode: "server" as const,
    sorting,
    handleSortingChange,
    rowSelection,
    setRowSelection,
    statusFilter: statusFilter as ScanStatusFilter,
    handleStatusFilterChange,
    selectedScans: actionSelectedScans,
    deleteDialogOpen,
    setDeleteDialogOpen,
    scanToDelete,
    bulkDeleteDialogOpen,
    setBulkDeleteDialogOpen,
    stopDialogOpen,
    setStopDialogOpen,
    scanToStop,
    progressDialogOpen,
    setProgressDialogOpen,
    progressData,
    runtimeDetailOpen,
    setRuntimeDetailOpen,
    scanForRuntimeDetail,
    handleDeleteScan,
    confirmDelete,
    handleBulkDelete,
    confirmBulkDelete,
    handleStopScan,
    confirmStop,
    selectedRowActions,
    batchStopDialogOpen,
    setBatchStopDialogOpen,
    activeSelectedScans,
    terminalSelectedCount,
    canBatchStop,
    batchStopMutationPending,
    handleBatchStop,
    confirmBatchStop,
    handleViewProgress,
    handleViewRuntimeDetail,
  }
}

export type ScanHistoryListViewState = ReturnType<typeof useScanHistoryListViewState>
