import React from "react"
import { useLocale, useTranslations } from "next-intl"
import type { SortingState } from "@tanstack/react-table"

import { useTarget } from "@/hooks/use-targets"
import {
  useBatchDeleteSubdomains,
  useExportSubdomains,
  useTargetSubdomains,
  useScanSubdomains,
} from "@/hooks/use-subdomains"
import {
  applyBusinessListControlChange,
  compileBusinessListFilter,
  compileBusinessListOrderBy,
  createBusinessListQuery,
  getCurrentCursorNextPageToken,
  getCursorPageTransition,
  getCursorPaginationNavigation,
  setBusinessListPage,
  toggleBusinessListSorting,
  type BusinessListFilterCompilerConfig,
  type BusinessListSortableFieldConfig,
  type BusinessListSorting,
} from "@/components/shared/data-table/business-list-query"
import { useCursorPaginationScopeChange } from "@/components/shared/data-table/use-cursor-pagination-scope"
import { getDateLocale } from "@/lib/date-utils"
import { escapeCSV, formatDateForCSV } from "@/lib/csv-utils"
import { saveBlobAsFile } from "@/lib/file-save-utils"
import { createSubdomainColumns } from "./subdomains-columns"

import type { Subdomain } from "@/types/subdomain.types"
import type { CursorPaginationSummary } from "@/types/data-table.types"

const SUBDOMAIN_FILTER_FIELDS: BusinessListFilterCompilerConfig = {
  search: { field: "dnsName", operator: "=" },
  facets: {},
}

const SUBDOMAIN_SORTABLE_FIELDS: Record<string, BusinessListSortableFieldConfig> = {
  dnsName: { orderBy: "dnsName", firstDirection: "asc" },
  createdAt: { orderBy: "createdAt", firstDirection: "desc" },
}

const SUBDOMAIN_DEFAULT_SORTING: BusinessListSorting = { field: "createdAt", direction: "desc" }

function subdomainQueryFieldToColumnId(field: string) {
  return field === "dnsName" ? "name" : field
}

function subdomainColumnIdToQueryField(columnId: string) {
  return columnId === "name" ? "dnsName" : columnId
}

interface SubdomainsDetailViewStateOptions {
  targetId?: number
  scanId?: number
}

export function useSubdomainsDetailViewState({
  targetId,
  scanId,
}: SubdomainsDetailViewStateOptions) {
  const [selectedSubdomains, setSelectedSubdomains] = React.useState<Subdomain[]>([])
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [isDeleting, setIsDeleting] = React.useState(false)
  const [bulkAddOpen, setBulkAddOpen] = React.useState(false)
  const [query, setQuery] = React.useState(() => createBusinessListQuery({
    pageSize: 10,
    sorting: SUBDOMAIN_DEFAULT_SORTING,
  }))
  const [pageTokens, setPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })
  const [scanQuery, setScanQuery] = React.useState(() => createBusinessListQuery({
    pageSize: 10,
    sorting: SUBDOMAIN_DEFAULT_SORTING,
  }))
  const [scanPageTokens, setScanPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })
  const cursorScopeKey = targetId ? `target:${targetId}` : `scan:${scanId ?? ""}`
  const hasCursorScopeChanged = useCursorPaginationScopeChange(cursorScopeKey)

  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const tSubdomains = useTranslations("subdomains")
  const tToast = useTranslations("toast")
  const locale = useLocale()

  const translations = React.useMemo(
    () => ({
      columns: {
        subdomain: tColumns("subdomain.subdomain"),
        createdAt: tColumns("common.createdAt"),
      },
      actions: {
        selectAll: tCommon("actions.selectAll"),
        selectRow: tCommon("actions.selectRow"),
      },
    }),
    [tColumns, tCommon]
  )

  const page = query.pageIndex ?? 1
  const pageSize = query.pageSize
  const scanPage = scanQuery.pageIndex ?? 1
  const scanPageSize = scanQuery.pageSize
  const activeTargetPage = hasCursorScopeChanged ? 1 : page
  const activeScanPage = hasCursorScopeChanged ? 1 : scanPage
  const targetPageToken = hasCursorScopeChanged ? undefined : query.pageToken
  const scanPageToken = hasCursorScopeChanged ? undefined : scanQuery.pageToken
  const targetSearchQuery = query.search ?? ""
  const scanSearchQuery = scanQuery.search ?? ""
  const targetPagination = React.useMemo(
    () => ({ pageIndex: activeTargetPage - 1, pageSize }),
    [activeTargetPage, pageSize]
  )
  const scanPagination = React.useMemo(
    () => ({ pageIndex: activeScanPage - 1, pageSize: scanPageSize }),
    [activeScanPage, scanPageSize]
  )
  const targetSorting = React.useMemo<SortingState>(() => {
    if (!query.sorting) return []
    return [{
      id: subdomainQueryFieldToColumnId(query.sorting.field),
      desc: query.sorting.direction === "desc",
    }]
  }, [query.sorting])
  const scanSorting = React.useMemo<SortingState>(() => {
    if (!scanQuery.sorting) return []
    return [{
      id: subdomainQueryFieldToColumnId(scanQuery.sorting.field),
      desc: scanQuery.sorting.direction === "desc",
    }]
  }, [scanQuery.sorting])

  const compiledFilter = compileBusinessListFilter(
    { search: query.search, filters: query.filters },
    SUBDOMAIN_FILTER_FIELDS
  )
  const compiledOrderBy = compileBusinessListOrderBy(query.sorting, SUBDOMAIN_SORTABLE_FIELDS)
  const compiledScanFilter = compileBusinessListFilter(
    { search: scanQuery.search, filters: scanQuery.filters },
    SUBDOMAIN_FILTER_FIELDS
  )
  const compiledScanOrderBy = compileBusinessListOrderBy(scanQuery.sorting, SUBDOMAIN_SORTABLE_FIELDS)

  const targetSubdomainsQuery = useTargetSubdomains(
    targetId || 0,
    {
      pageSize,
      pageToken: targetPageToken,
      filter: compiledFilter,
      orderBy: compiledOrderBy,
    },
    { enabled: !!targetId }
  )

  const scanSubdomainsQuery = useScanSubdomains(
    scanId || 0,
    {
      pageSize: scanPageSize,
      pageToken: scanPageToken,
      filter: compiledScanFilter,
      orderBy: compiledScanOrderBy,
    },
    { enabled: !!scanId }
  )

  const activeQuery = targetId ? targetSubdomainsQuery : scanSubdomainsQuery
  const { data: subdomainsData, isLoading, isFetching, error, refetch } = activeQuery
  const targetNextPageToken = getCurrentCursorNextPageToken(
    targetSubdomainsQuery.data?.nextPageToken,
    targetSubdomainsQuery.isPlaceholderData || hasCursorScopeChanged,
  )
  const scanNextPageToken = getCurrentCursorNextPageToken(
    scanSubdomainsQuery.data?.nextPageToken,
    scanSubdomainsQuery.isPlaceholderData || hasCursorScopeChanged,
  )
  const activeNextPageToken = targetId ? targetNextPageToken : scanNextPageToken
  const bulkDeleteSubdomains = useBatchDeleteSubdomains()
  const exportSubdomains = useExportSubdomains({ targetId, scanId })

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

  const resetPaging = React.useCallback(() => {
    setPageTokens({ 1: undefined })
  }, [])
  const resetScanPaging = React.useCallback(() => {
    setScanPageTokens({ 1: undefined })
  }, [])

  React.useEffect(() => {
    if (!hasCursorScopeChanged) return

    setSelectedSubdomains([])
    setDeleteDialogOpen(false)
    setBulkAddOpen(false)
    resetPaging()
    resetScanPaging()
    setQuery((current) => setBusinessListPage(current, { pageIndex: 1, pageToken: undefined }))
    setScanQuery((current) => setBusinessListPage(current, { pageIndex: 1, pageToken: undefined }))
  }, [hasCursorScopeChanged, resetPaging, resetScanPaging])

  const commitTargetSearch = React.useCallback((value: string) => {
    const normalizedSearch = value.trim()
    if ((query.search ?? "") === normalizedSearch) {
      return
    }
    resetPaging()
    setQuery((current) => applyBusinessListControlChange(current, {
      search: normalizedSearch || undefined,
    }))
  }, [query.search, resetPaging])

  const commitScanSearch = React.useCallback((value: string) => {
    const normalizedSearch = value.trim()
    if ((scanQuery.search ?? "") === normalizedSearch) {
      return
    }
    resetScanPaging()
    setScanQuery((current) => applyBusinessListControlChange(current, {
      search: normalizedSearch || undefined,
    }))
  }, [resetScanPaging, scanQuery.search])

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
      if (!nextColumnId) {
        resetScanPaging()
        setScanQuery((current) => applyBusinessListControlChange(current, { sorting: SUBDOMAIN_DEFAULT_SORTING }))
        return
      }

      resetScanPaging()
      setScanQuery((current) => toggleBusinessListSorting(
        current,
        subdomainColumnIdToQueryField(nextColumnId),
        SUBDOMAIN_SORTABLE_FIELDS,
        SUBDOMAIN_DEFAULT_SORTING
      ))
      return
    }

    if (!nextColumnId) {
      resetPaging()
      setQuery((current) => applyBusinessListControlChange(current, { sorting: SUBDOMAIN_DEFAULT_SORTING }))
      return
    }

    resetPaging()
    setQuery((current) => toggleBusinessListSorting(
      current,
      subdomainColumnIdToQueryField(nextColumnId),
      SUBDOMAIN_SORTABLE_FIELDS,
      SUBDOMAIN_DEFAULT_SORTING
    ))
  }, [resetPaging, resetScanPaging, targetId])

  const cursorPaginationSummary: CursorPaginationSummary = {
    total: subdomainsData?.totalSize ?? subdomainsData?.total ?? subdomainsData?.results?.length ?? 0,
  }
  const paginationNavigation = getCursorPaginationNavigation({
    currentPage: targetId ? activeTargetPage : activeScanPage,
    pageTokens: hasCursorScopeChanged ? { 1: undefined } : targetId ? pageTokens : scanPageTokens,
    nextPageToken: activeNextPageToken,
  })

  const { data: targetData } = useTarget(targetId || 0, { enabled: !!targetId })

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

  const generateCSV = React.useCallback((items: Subdomain[]): string => {
    const bom = "\ufeff"
    const headers = ["name", "created_at"]

    const rows = items.map((item) =>
      [escapeCSV(item.name), escapeCSV(formatDateForCSV(item.createdAt))].join(",")
    )

    return bom + [headers.join(","), ...rows].join("\n")
  }, [])

  const subdomains = React.useMemo(() => {
    if (!subdomainsData?.results) return []
    return subdomainsData.results.map((item) => ({
      id: item.id,
      name: item.name,
      dnsName: item.name,
      createdAt: item.createdAt,
    }))
  }, [subdomainsData])

  const handleExportAll = React.useCallback(async () => {
    try {
      let blob: Blob | null = null

      if (scanId || targetId) {
        blob = await exportSubdomains()
      } else if (subdomains.length > 0) {
        const csvContent = generateCSV(subdomains)
        blob = new Blob([csvContent], { type: "text/csv;charset=utf-8" })
      }

      if (!blob) return

      const prefix = scanId ? `scan-${scanId}` : targetId ? `target-${targetId}` : "subdomains"
      saveBlobAsFile(blob, `${prefix}-subdomains-${Date.now()}.csv`)
    } catch (error) {
      void error
    }
  }, [exportSubdomains, generateCSV, scanId, subdomains, targetId])

  const handleExportSelected = React.useCallback(() => {
    if (selectedSubdomains.length === 0) {
      return
    }
    const csvContent = generateCSV(selectedSubdomains)
    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8" })
    saveBlobAsFile(blob, `subdomains-selected-${scanId ?? targetId ?? "all"}-${Date.now()}.csv`)
  }, [generateCSV, scanId, selectedSubdomains, targetId])

  const handleBulkDelete = React.useCallback(async () => {
    if (selectedSubdomains.length === 0) return

    setIsDeleting(true)
    try {
      const ids = selectedSubdomains.map((subdomain) => subdomain.id)
      await bulkDeleteSubdomains.mutateAsync({ targetId: targetId!, ids })
      setSelectedSubdomains([])
      setDeleteDialogOpen(false)
    } catch {
      void tToast
    } finally {
      setIsDeleting(false)
    }
  }, [bulkDeleteSubdomains, selectedSubdomains, targetId, tToast])

  const subdomainColumns = React.useMemo(
    () =>
      createSubdomainColumns({
        formatDate,
        t: translations,
      }),
    [formatDate, translations]
  )

  return {
    tCommon,
    tSubdomains,
    targetId,
    targetData,
    subdomainsData,
    isLoading,
    error,
    refetch,
    subdomains,
    subdomainColumns,
    selectedSubdomains,
    setSelectedSubdomains,
    deleteDialogOpen,
    setDeleteDialogOpen,
    isDeleting,
    bulkAddOpen,
    setBulkAddOpen,
    searchQuery: targetId ? targetSearchQuery : scanSearchQuery,
    commitSearch: targetId ? commitTargetSearch : commitScanSearch,
    isSearching: isFetching,
    pagination: targetId ? targetPagination : scanPagination,
    cursorPaginationSummary,
    paginationNavigation,
    handlePaginationChange,
    sortingMode: "server" as const,
    sorting: targetId ? targetSorting : scanSorting,
    handleSortingChange,
    handleExportAll,
    handleExportSelected,
    handleBulkDelete,
  }
}

export type SubdomainsDetailViewState = ReturnType<typeof useSubdomainsDetailViewState>
