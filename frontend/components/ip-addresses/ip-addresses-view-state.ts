import React from "react"
import { useLocale, useTranslations } from "next-intl"
import type { SortingState } from "@tanstack/react-table"
import { toastFeedback } from "@/lib/toast-helpers"

import {
  useBulkDeleteIPAddresses,
  useExportIPAddresses,
  useTargetIPAddresses,
  useScanIPAddresses,
  useTargetPortOptions,
  useScanPortOptions,
} from "@/hooks/use-ip-addresses"
import {
  applyBusinessListControlChange,
  compileBusinessListOrderBy,
  createBusinessListQuery,
  getCurrentCursorNextPageToken,
  getCursorPageTransition,
  getCursorPaginationNavigation,
  setBusinessListPage,
  toggleBusinessListSorting,
  type BusinessListQuery,
  type BusinessListSortableFieldConfig,
  type BusinessListSorting,
} from "@/components/shared/data-table/business-list-query"
import { useCursorPaginationScopeChange } from "@/components/shared/data-table/use-cursor-pagination-scope"
import { getDateLocale } from "@/lib/date-utils"
import { escapeCSV, formatDateForCSV } from "@/lib/csv-utils"
import { saveBlobAsFile } from "@/lib/file-save-utils"
import { createIPAddressColumns } from "./ip-addresses-columns"
import { composeWebsiteScopeFilter } from "@/lib/website-scope"

import type { IPAddress } from "@/types/ip-address.types"
import type { WebsiteAssetScope } from "@/types/website.types"
import type { CursorPaginationSummary } from "@/types/data-table.types"

const IP_ADDRESS_SORTABLE_FIELDS: Record<string, BusinessListSortableFieldConfig> = {
  ip: { orderBy: "ip", firstDirection: "asc" },
  createdAt: { orderBy: "createdAt", firstDirection: "desc" },
}

const IP_ADDRESS_DEFAULT_SORTING: BusinessListSorting = { field: "createdAt", direction: "desc" }

function quoteFilterValue(value: string) {
  return value.replace(/\\/g, "\\\\").replace(/"/g, "\\\"")
}

function uniqueNonEmptyValues(values: string[] | undefined) {
  return Array.from(new Set((values ?? []).map((value) => value.trim()).filter(Boolean)))
}

function compileIPAddressFilter(query: Pick<BusinessListQuery, "search" | "filters">) {
  const clauses: string[] = []
  const search = query.search?.trim()
  if (search) {
    const value = quoteFilterValue(search)
    clauses.push(`(ip="${value}" || host="${value}")`)
  }

  const portPredicates = uniqueNonEmptyValues(query.filters?.port).map(
    (port) => `port="${quoteFilterValue(port)}"`
  )
  if (portPredicates.length === 1) {
    clauses.push(portPredicates[0]!)
  } else if (portPredicates.length > 1) {
    clauses.push(`(${portPredicates.join(" || ")})`)
  }

  return clauses.length > 0 ? clauses.join(" && ") : undefined
}

interface IPAddressesViewStateOptions {
  targetId?: number
  scanId?: number
  websiteScope?: WebsiteAssetScope
}

export function useIPAddressesViewState({ targetId, scanId, websiteScope }: IPAddressesViewStateOptions) {
  const [selectedIPAddresses, setSelectedIPAddresses] = React.useState<IPAddress[]>([])
  const [portFilter, setPortFilter] = React.useState<string[]>([])
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [isDeleting, setIsDeleting] = React.useState(false)
  const [query, setQuery] = React.useState(() => createBusinessListQuery({
    pageSize: 10,
    sorting: IP_ADDRESS_DEFAULT_SORTING,
  }))
  const [pageTokens, setPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })
  const [scanQuery, setScanQuery] = React.useState(() => createBusinessListQuery({
    pageSize: 10,
    sorting: IP_ADDRESS_DEFAULT_SORTING,
  }))
  const [scanPageTokens, setScanPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })

  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const tTooltips = useTranslations("tooltips")
  const tToast = useTranslations("toast")
  const tStatus = useTranslations("common.status")
  const locale = useLocale()
  const bulkDeleteIPAddresses = useBulkDeleteIPAddresses()
  const exportIPAddresses = useExportIPAddresses({ targetId, scanId })
  const isReadOnly = websiteScope?.readOnly === true
  const websiteScopeKey = websiteScope ? `${websiteScope.host}\u0000${websiteScope.url}` : ""
  const cursorScopeKey = targetId
    ? `target:${targetId}\u0000${websiteScopeKey}`
    : `scan:${scanId ?? ""}`
  const hasCursorScopeChanged = useCursorPaginationScopeChange(cursorScopeKey)
  const ipAddressLabel = isReadOnly
    ? tColumns("ipAddress.resolvedIpAddress")
    : tColumns("ipAddress.ipAddress")
  const searchPlaceholder = isReadOnly
    ? tCommon("actions.searchResolvedIPOrHost")
    : tCommon("actions.searchIPOrHost")

  const page = query.pageIndex ?? 1
  const pageSize = query.pageSize
  const scanPage = scanQuery.pageIndex ?? 1
  const scanPageSize = scanQuery.pageSize
  const activeTargetPage = hasCursorScopeChanged ? 1 : page
  const activeScanPage = hasCursorScopeChanged ? 1 : scanPage
  const targetPageToken = hasCursorScopeChanged ? undefined : query.pageToken
  const scanPageToken = hasCursorScopeChanged ? undefined : scanQuery.pageToken
  const filterQuery = targetId ? (query.search ?? "") : (scanQuery.search ?? "")
  const pagination = React.useMemo(
    () => ({ pageIndex: (targetId ? activeTargetPage : activeScanPage) - 1, pageSize: targetId ? pageSize : scanPageSize }),
    [activeScanPage, activeTargetPage, pageSize, scanPageSize, targetId]
  )
  const sorting = React.useMemo<SortingState>(() => {
    const currentSorting = targetId ? query.sorting : scanQuery.sorting
    if (!currentSorting) return []
    return [{ id: currentSorting.field, desc: currentSorting.direction === "desc" }]
  }, [query.sorting, scanQuery.sorting, targetId])

  const compiledFilter = compileIPAddressFilter({ search: query.search, filters: query.filters })
  const scopedTargetFilter = websiteScope
    ? composeWebsiteScopeFilter(websiteScope, "host", compiledFilter)
    : compiledFilter
  const compiledOrderBy = compileBusinessListOrderBy(query.sorting, IP_ADDRESS_SORTABLE_FIELDS)
  const compiledScanFilter = compileIPAddressFilter({ search: scanQuery.search, filters: scanQuery.filters })
  const compiledScanOrderBy = compileBusinessListOrderBy(scanQuery.sorting, IP_ADDRESS_SORTABLE_FIELDS)

  const translations = React.useMemo(
    () => ({
      columns: {
        ipAddress: ipAddressLabel,
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
    }),
    [ipAddressLabel, tCommon, tTooltips]
  )

  const resetPaging = React.useCallback(() => {
    setPageTokens({ 1: undefined })
  }, [])
  const resetScanPaging = React.useCallback(() => {
    setScanPageTokens({ 1: undefined })
  }, [])

  React.useEffect(() => {
    if (!hasCursorScopeChanged) return

    setSelectedIPAddresses([])
    resetPaging()
    resetScanPaging()
    setQuery((current) => setBusinessListPage(current, { pageIndex: 1, pageToken: undefined }))
    setScanQuery((current) => setBusinessListPage(current, { pageIndex: 1, pageToken: undefined }))
  }, [hasCursorScopeChanged, resetPaging, resetScanPaging])

  const commitFilterSearch = React.useCallback((value: string) => {
    const normalizedSearch = value.trim()
    if (targetId) {
      if ((query.search ?? "") === normalizedSearch) return
      resetPaging()
      setQuery((current) => applyBusinessListControlChange(current, {
        search: normalizedSearch || undefined,
      }))
      return
    }

    if ((scanQuery.search ?? "") === normalizedSearch) return
    resetScanPaging()
    setScanQuery((current) => applyBusinessListControlChange(current, {
      search: normalizedSearch || undefined,
    }))
  }, [query.search, resetPaging, resetScanPaging, scanQuery.search, targetId])

  const handlePortFilterChange = React.useCallback((values: string[]) => {
    const nextValues = uniqueNonEmptyValues(values)
    setPortFilter(nextValues)
    if (targetId) {
      resetPaging()
      setQuery((current) => applyBusinessListControlChange(current, {
        filters: { ...current.filters, port: nextValues },
      }))
      return
    }

    resetScanPaging()
    setScanQuery((current) => applyBusinessListControlChange(current, {
      filters: { ...current.filters, port: nextValues },
    }))
  }, [resetPaging, resetScanPaging, targetId])

  const targetQuery = useTargetIPAddresses(
    targetId || 0,
    {
      pageSize,
      pageToken: targetPageToken,
      filter: scopedTargetFilter,
      orderBy: compiledOrderBy,
    },
    { enabled: !!targetId, websiteScope }
  )

  const scanIPAddressesQuery = useScanIPAddresses(
    scanId || 0,
    {
      pageSize: scanPageSize,
      pageToken: scanPageToken,
      filter: compiledScanFilter,
      orderBy: compiledScanOrderBy,
    },
    { enabled: !!scanId }
  )
  const targetPortOptionsQuery = useTargetPortOptions(targetId || 0, { enabled: !!targetId })
  const scanPortOptionsQuery = useScanPortOptions(scanId || 0, { enabled: !targetId && !!scanId })

  const activeQuery = targetId ? targetQuery : scanIPAddressesQuery
  const activePortOptionsQuery = targetId ? targetPortOptionsQuery : scanPortOptionsQuery
  const { data, isLoading, isFetching, error, refetch } = activeQuery
  const targetNextPageToken = getCurrentCursorNextPageToken(
    targetQuery.data?.nextPageToken,
    targetQuery.isPlaceholderData || hasCursorScopeChanged,
  )
  const scanNextPageToken = getCurrentCursorNextPageToken(
    scanIPAddressesQuery.data?.nextPageToken,
    scanIPAddressesQuery.isPlaceholderData || hasCursorScopeChanged,
  )
  const activeNextPageToken = targetId ? targetNextPageToken : scanNextPageToken

  React.useEffect(() => {
    if (targetNextPageToken) {
      setPageTokens((tokens) => ({ ...tokens, [activeTargetPage + 1]: targetNextPageToken }))
    }
  }, [activeTargetPage, targetNextPageToken])

  React.useEffect(() => {
    if (scanNextPageToken) {
      setScanPageTokens((tokens) => ({ ...tokens, [activeScanPage + 1]: scanNextPageToken }))
    }
  }, [activeScanPage, scanNextPageToken])

  const handlePaginationChange = React.useCallback(
    (newPagination: { pageIndex: number; pageSize: number }) => {
      if (!targetId) {
        const nextPage = newPagination.pageIndex + 1
        if (newPagination.pageSize !== scanPageSize) {
          resetScanPaging()
          setScanQuery((current) => applyBusinessListControlChange(current, { pageSize: newPagination.pageSize }))
          return
        }

        if (nextPage === 1) {
          resetScanPaging()
          setScanQuery((current) => setBusinessListPage(current, { pageIndex: 1, pageToken: undefined }))
          return
        }

        const transition = getCursorPageTransition({
          currentPage: activeScanPage,
          pageTokens: scanPageTokens,
          nextPageToken: scanNextPageToken,
          requestedPage: nextPage,
        })
        if (!transition.reachable) return
        setScanQuery((current) => setBusinessListPage(current, {
          pageIndex: nextPage,
          pageToken: transition.pageToken,
        }))
        return
      }

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
        currentPage: activeTargetPage,
        pageTokens,
        nextPageToken: targetNextPageToken,
        requestedPage: nextPage,
      })
      if (!transition.reachable) return
      setQuery((current) => setBusinessListPage(current, {
        pageIndex: nextPage,
        pageToken: transition.pageToken,
      }))
    },
    [activeScanPage, activeTargetPage, pageSize, pageTokens, resetPaging, resetScanPaging, scanNextPageToken, scanPageSize, scanPageTokens, targetId, targetNextPageToken]
  )

  const handleSortingChange = React.useCallback((nextSorting: SortingState) => {
    const nextColumnId = nextSorting[0]?.id
    if (!targetId) {
      resetScanPaging()
      if (!nextColumnId) {
        setScanQuery((current) => applyBusinessListControlChange(current, { sorting: IP_ADDRESS_DEFAULT_SORTING }))
        return
      }
      setScanQuery((current) => toggleBusinessListSorting(
        current,
        nextColumnId,
        IP_ADDRESS_SORTABLE_FIELDS,
        IP_ADDRESS_DEFAULT_SORTING
      ))
      return
    }

    resetPaging()
    if (!nextColumnId) {
      setQuery((current) => applyBusinessListControlChange(current, { sorting: IP_ADDRESS_DEFAULT_SORTING }))
      return
    }
    setQuery((current) => toggleBusinessListSorting(
      current,
      nextColumnId,
      IP_ADDRESS_SORTABLE_FIELDS,
      IP_ADDRESS_DEFAULT_SORTING
    ))
  }, [resetPaging, resetScanPaging, targetId])

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
      createIPAddressColumns({
        formatDate,
        t: translations,
        includeSelection: !isReadOnly,
      }),
    [formatDate, isReadOnly, translations]
  )

  const ipAddresses: IPAddress[] = React.useMemo(() => {
    return data?.results ?? []
  }, [data])

  const cursorPaginationSummary: CursorPaginationSummary = {
    total: data?.totalSize ?? data?.total ?? ipAddresses.length,
  }
  const paginationNavigation = getCursorPaginationNavigation({
    currentPage: targetId ? activeTargetPage : activeScanPage,
    pageTokens: hasCursorScopeChanged ? { 1: undefined } : targetId ? pageTokens : scanPageTokens,
    nextPageToken: activeNextPageToken,
  })

  const handleSelectionChange = React.useCallback((selectedRows: IPAddress[]) => {
    setSelectedIPAddresses(selectedRows)
  }, [])

  const generateCSV = React.useCallback((items: IPAddress[]): string => {
    const bom = "\ufeff"
    const headers = ["ip", "host", "port", "created_at"]

    const rows: string[] = []
    for (const item of items) {
      for (const host of item.hosts) {
        for (const port of item.ports) {
          rows.push(
            [
              escapeCSV(item.ip),
              escapeCSV(host),
              escapeCSV(String(port)),
              escapeCSV(formatDateForCSV(item.createdAt)),
            ].join(",")
          )
        }
      }
    }

    return bom + [headers.join(","), ...rows].join("\n")
  }, [])

  const handleExportAll = React.useCallback(async () => {
    try {
      let blob: Blob | null = null

      if (scanId || targetId) {
        blob = await exportIPAddresses()
      } else if (ipAddresses.length > 0) {
        const csvContent = generateCSV(ipAddresses)
        blob = new Blob([csvContent], { type: "text/csv;charset=utf-8" })
      }

      if (!blob) return

      const prefix = scanId ? `scan-${scanId}` : targetId ? `target-${targetId}` : "ip-addresses"
      saveBlobAsFile(blob, `${prefix}-ip-addresses-${Date.now()}.csv`)
    } catch {
      toastFeedback.error(tToast("exportFailed"))
    }
  }, [exportIPAddresses, generateCSV, ipAddresses, scanId, targetId, tToast])

  const handleExportSelected = React.useCallback(async () => {
    if (selectedIPAddresses.length === 0) {
      return
    }

    try {
      const ips = selectedIPAddresses.map((item) => item.ip)
      let blob: Blob | null = null

      if (targetId) {
        blob = await exportIPAddresses(ips)
      } else {
        const csvContent = generateCSV(selectedIPAddresses)
        blob = new Blob([csvContent], { type: "text/csv;charset=utf-8" })
      }

      if (!blob) return

      const prefix = scanId ? `scan-${scanId}` : targetId ? `target-${targetId}` : "ip-addresses"
      saveBlobAsFile(blob, `${prefix}-ip-addresses-selected-${Date.now()}.csv`)
    } catch {
      toastFeedback.error(tToast("exportFailed"))
    }
  }, [exportIPAddresses, generateCSV, scanId, selectedIPAddresses, targetId, tToast])

  const handleBulkDelete = React.useCallback(async () => {
    if (selectedIPAddresses.length === 0) return

    setIsDeleting(true)
    try {
      const ips = selectedIPAddresses.map((item) => item.ip)
      await bulkDeleteIPAddresses.mutateAsync(ips)
      setSelectedIPAddresses([])
      setDeleteDialogOpen(false)
    } catch {
      void tToast
    } finally {
      setIsDeleting(false)
    }
  }, [bulkDeleteIPAddresses, selectedIPAddresses, tToast])

  return {
    tCommon,
    tStatus,
    targetId,
    isReadOnly,
    data,
    error,
    isLoading,
    isSearching: isFetching,
    refetch,
    columns,
    ipAddresses,
    filterQuery,
    searchPlaceholder,
    commitFilterSearch,
    portFilter,
    portOptions: activePortOptionsQuery.data?.results ?? [],
    handlePortFilterChange,
    pagination,
    cursorPaginationSummary,
    paginationNavigation,
    handlePaginationChange,
    sortingMode: "server" as const,
    sorting,
    handleSortingChange,
    handleSelectionChange,
    handleExportAll,
    handleExportSelected,
    handleBulkDelete,
    deleteDialogOpen,
    setDeleteDialogOpen,
    selectedIPAddresses,
    isDeleting,
  }
}

export type IPAddressesViewState = ReturnType<typeof useIPAddressesViewState>
