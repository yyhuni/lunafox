import React from "react"
import { useLocale, useTranslations } from "next-intl"
import type { SortingState, VisibilityState } from "@tanstack/react-table"
import { toastFeedback } from "@/lib/toast-helpers"

import { useTargetEndpoints, useTarget } from "@/hooks/use-targets"
import {
  useBatchDeleteEndpoints,
  useDeleteEndpoint,
  useScanEndpointFilterOptions,
  useExportEndpoints,
  useTargetEndpointFilterOptions,
  useScanEndpoints,
} from "@/hooks/use-endpoints"
import {
  applyBusinessListControlChange,
  compileBusinessListFilter,
  compileBusinessListOrderBy,
  createBusinessListQuery,
  getCurrentCursorNextPageToken,
  getCursorPageTransition,
  getCursorPaginationNavigation,
  preserveRawURLSearchInput,
  setBusinessListPage,
  toggleBusinessListSorting,
  type BusinessListFilterCompilerConfig,
  type BusinessListSortableFieldConfig,
  type BusinessListSorting,
} from "@/components/shared/data-table/business-list-query"
import { useCursorPaginationScopeChange } from "@/components/shared/data-table/use-cursor-pagination-scope"
import { getDateLocale } from "@/lib/date-utils"
import { escapeCSV, formatArrayForCSV, formatDateForCSV } from "@/lib/csv-utils"
import { saveBlobAsFile } from "@/lib/file-save-utils"
import { createEndpointColumns } from "./endpoints-columns"
import { composeWebsiteScopeFilter } from "@/lib/website-scope"

import type { Endpoint, EndpointFilterOption, EndpointFilterOptionField } from "@/types/endpoint.types"
import type { WebsiteAssetScope } from "@/types/website.types"
import type { CursorPaginationSummary } from "@/types/data-table.types"

function useScopedEndpointFilterOptions(
  field: EndpointFilterOptionField,
  { targetId, scanId }: EndpointsDetailViewStateOptions
) {
  const targetOptions = useTargetEndpointFilterOptions(targetId || 0, field, { enabled: !!targetId })
  const scanOptions = useScanEndpointFilterOptions(scanId || 0, field, { enabled: !targetId && !!scanId })
  return targetId ? targetOptions : scanOptions
}

function normalizeFilterOptions(data: { results?: EndpointFilterOption[] } | undefined) {
  return data?.results ?? []
}

const ENDPOINT_SORTABLE_FIELDS: Record<string, BusinessListSortableFieldConfig> = {
  statusCode: { orderBy: "statusCode", firstDirection: "asc" },
  contentLength: { orderBy: "contentLength", firstDirection: "desc" },
  createdAt: { orderBy: "createdAt", firstDirection: "desc" },
}

const ENDPOINT_DEFAULT_SORTING: BusinessListSorting = { field: "createdAt", direction: "desc" }
export const DEFAULT_ENDPOINT_COLUMN_VISIBILITY: VisibilityState = {
  host: false,
  location: false,
  responseBody: false,
  responseHeaders: false,
}

const ENDPOINT_FILTER_CONFIG: BusinessListFilterCompilerConfig = {
  search: { field: "url", operator: "==" },
  facets: {
    statusCode: { field: "statusCode" },
    tech: { field: "tech" },
    webserver: { field: "webserver" },
    contentType: { field: "contentType" },
    vhost: { field: "vhost" },
  },
}

function uniqueNonEmptyValues(values: string[] | undefined) {
  return Array.from(new Set((values ?? []).map((value) => value.trim()).filter(Boolean)))
}

interface EndpointsDetailViewStateOptions {
  targetId?: number
  scanId?: number
  websiteScope?: WebsiteAssetScope
}

export function useEndpointsDetailViewState({ targetId, scanId, websiteScope }: EndpointsDetailViewStateOptions) {
  const [columnVisibility, setColumnVisibility] = React.useState<VisibilityState>(
    DEFAULT_ENDPOINT_COLUMN_VISIBILITY
  )
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [endpointToDelete, setEndpointToDelete] = React.useState<Endpoint | null>(null)
  const [selectedEndpoints, setSelectedEndpoints] = React.useState<Endpoint[]>([])
  const [bulkAddDialogOpen, setBulkAddDialogOpen] = React.useState(false)
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = React.useState(false)
  const [isDeleting, setIsDeleting] = React.useState(false)
  const [query, setQuery] = React.useState(() => createBusinessListQuery({
    pageSize: 10,
    sorting: ENDPOINT_DEFAULT_SORTING,
  }))
  const [pageTokens, setPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })
  const [scanQuery, setScanQuery] = React.useState(() => createBusinessListQuery({
    pageSize: 10,
    sorting: ENDPOINT_DEFAULT_SORTING,
  }))
  const [scanPageTokens, setScanPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })

  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const tToast = useTranslations("toast")
  const tConfirm = useTranslations("common.confirm")
  const locale = useLocale()

  const translations = React.useMemo(
    () => ({
      columns: {
        url: tColumns("common.url"),
        host: tColumns("endpoint.host"),
        title: tColumns("endpoint.title"),
        status: tColumns("common.status"),
        contentLength: tColumns("endpoint.contentLength"),
        location: tColumns("endpoint.location"),
        webServer: tColumns("endpoint.webServer"),
        contentType: tColumns("endpoint.contentType"),
        technologies: tColumns("endpoint.technologies"),
        responseBody: tColumns("endpoint.responseBody"),
        vhost: tColumns("endpoint.vhost"),
        responseHeaders: tColumns("endpoint.responseHeaders"),
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
  const deleteEndpoint = useDeleteEndpoint()
  const batchDeleteEndpoints = useBatchDeleteEndpoints()
  const exportEndpoints = useExportEndpoints({ targetId, scanId })
  const isReadOnly = websiteScope?.readOnly === true
  const websiteScopeKey = websiteScope ? `${websiteScope.host}\u0000${websiteScope.url}` : ""
  const cursorScopeKey = targetId
    ? `target:${targetId}\u0000${websiteScopeKey}`
    : `scan:${scanId ?? ""}`
  const hasCursorScopeChanged = useCursorPaginationScopeChange(cursorScopeKey)

  const page = query.pageIndex ?? 1
  const pageSize = query.pageSize
  const scanPage = scanQuery.pageIndex ?? 1
  const scanPageSize = scanQuery.pageSize
  const activeTargetPage = hasCursorScopeChanged ? 1 : page
  const activeScanPage = hasCursorScopeChanged ? 1 : scanPage
  const targetPageToken = hasCursorScopeChanged ? undefined : query.pageToken
  const scanPageToken = hasCursorScopeChanged ? undefined : scanQuery.pageToken
  const activeBusinessQuery = targetId ? query : scanQuery
  const filterQuery = activeBusinessQuery.search ?? ""
  const pagination = React.useMemo(
    () => ({ pageIndex: (targetId ? activeTargetPage : activeScanPage) - 1, pageSize: targetId ? pageSize : scanPageSize }),
    [activeScanPage, activeTargetPage, pageSize, scanPageSize, targetId]
  )
  const sorting = React.useMemo<SortingState>(() => {
    const currentSorting = activeBusinessQuery.sorting
    if (!currentSorting) return []
    return [{ id: currentSorting.field, desc: currentSorting.direction === "desc" }]
  }, [activeBusinessQuery.sorting])

  const compiledFilter = compileBusinessListFilter({ search: query.search, filters: query.filters }, ENDPOINT_FILTER_CONFIG)
  const scopedTargetFilter = websiteScope
    ? composeWebsiteScopeFilter(websiteScope, "websiteUrl", compiledFilter)
    : compiledFilter
  const compiledOrderBy = compileBusinessListOrderBy(query.sorting, ENDPOINT_SORTABLE_FIELDS)
  const compiledScanFilter = compileBusinessListFilter({ search: scanQuery.search, filters: scanQuery.filters }, ENDPOINT_FILTER_CONFIG)
  const compiledScanOrderBy = compileBusinessListOrderBy(scanQuery.sorting, ENDPOINT_SORTABLE_FIELDS)

  const resetPaging = React.useCallback(() => {
    setPageTokens({ 1: undefined })
  }, [])
  const resetScanPaging = React.useCallback(() => {
    setScanPageTokens({ 1: undefined })
  }, [])

  React.useEffect(() => {
    if (!hasCursorScopeChanged) return

    setSelectedEndpoints([])
    setBulkAddDialogOpen(false)
    setBulkDeleteDialogOpen(false)
    setDeleteDialogOpen(false)
    setEndpointToDelete(null)
    resetPaging()
    resetScanPaging()
    setQuery((current) => setBusinessListPage(current, { pageIndex: 1, pageToken: undefined }))
    setScanQuery((current) => setBusinessListPage(current, { pageIndex: 1, pageToken: undefined }))
  }, [hasCursorScopeChanged, resetPaging, resetScanPaging])

  const commitFilterSearch = React.useCallback((value: string) => {
    const normalizedSearch = preserveRawURLSearchInput(value)
    if (targetId) {
      if ((query.search ?? "") === (normalizedSearch ?? "")) return
      resetPaging()
      setQuery((current) => applyBusinessListControlChange(current, {
        search: normalizedSearch,
      }))
      return
    }

    if ((scanQuery.search ?? "") === (normalizedSearch ?? "")) return
    resetScanPaging()
    setScanQuery((current) => applyBusinessListControlChange(current, {
      search: normalizedSearch,
    }))
  }, [query.search, resetPaging, resetScanPaging, scanQuery.search, targetId])

  const updateEndpointFacet = React.useCallback((field: string, values: string[]) => {
    const nextValues = uniqueNonEmptyValues(values)
    if (targetId) {
      resetPaging()
      setQuery((current) => applyBusinessListControlChange(current, {
        filters: { ...current.filters, [field]: nextValues },
      }))
      return
    }

    resetScanPaging()
    setScanQuery((current) => applyBusinessListControlChange(current, {
      filters: { ...current.filters, [field]: nextValues },
    }))
  }, [resetPaging, resetScanPaging, targetId])

  const targetEndpointsQuery = useTargetEndpoints(
    targetId || 0,
    {
      pageSize,
      pageToken: targetPageToken,
      filter: scopedTargetFilter,
      orderBy: compiledOrderBy,
    },
    { enabled: !!targetId, websiteScope }
  )

  const scanEndpointsQuery = useScanEndpoints(
    scanId || 0,
    {
      pageSize: scanPageSize,
      pageToken: scanPageToken,
      filter: compiledScanFilter,
      orderBy: compiledScanOrderBy,
    },
    { enabled: !!scanId }
  )

  const statusCodeOptionsQuery = useScopedEndpointFilterOptions("statusCode", { targetId, scanId })
  const techOptionsQuery = useScopedEndpointFilterOptions("tech", { targetId, scanId })
  const webserverOptionsQuery = useScopedEndpointFilterOptions("webserver", { targetId, scanId })
  const contentTypeOptionsQuery = useScopedEndpointFilterOptions("contentType", { targetId, scanId })
  const vhostOptionsQuery = useScopedEndpointFilterOptions("vhost", { targetId, scanId })

  const activeQuery = targetId ? targetEndpointsQuery : scanEndpointsQuery
  const { data, isLoading, isFetching, error, refetch } = activeQuery
  const targetNextPageToken = getCurrentCursorNextPageToken(
    targetEndpointsQuery.data?.nextPageToken,
    targetEndpointsQuery.isPlaceholderData || hasCursorScopeChanged,
  )
  const scanNextPageToken = getCurrentCursorNextPageToken(
    scanEndpointsQuery.data?.nextPageToken,
    scanEndpointsQuery.isPlaceholderData || hasCursorScopeChanged,
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

  const confirmDelete = React.useCallback(() => {
    if (!endpointToDelete) return
    setDeleteDialogOpen(false)
    setEndpointToDelete(null)
    deleteEndpoint.mutate(endpointToDelete.id)
  }, [deleteEndpoint, endpointToDelete])

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
        setScanQuery((current) => applyBusinessListControlChange(current, { sorting: ENDPOINT_DEFAULT_SORTING }))
        return
      }
      setScanQuery((current) => toggleBusinessListSorting(
        current,
        nextColumnId,
        ENDPOINT_SORTABLE_FIELDS,
        ENDPOINT_DEFAULT_SORTING
      ))
      return
    }

    resetPaging()
    if (!nextColumnId) {
      setQuery((current) => applyBusinessListControlChange(current, { sorting: ENDPOINT_DEFAULT_SORTING }))
      return
    }
    setQuery((current) => toggleBusinessListSorting(
      current,
      nextColumnId,
      ENDPOINT_SORTABLE_FIELDS,
      ENDPOINT_DEFAULT_SORTING
    ))
  }, [resetPaging, resetScanPaging, targetId])

  const handleSelectionChange = React.useCallback((selectedRows: Endpoint[]) => {
    setSelectedEndpoints(selectedRows)
  }, [])

  const endpointColumns = React.useMemo(
    () =>
      createEndpointColumns({
        formatDate,
        t: translations,
        includeSelection: !isReadOnly,
      }),
    [formatDate, isReadOnly, translations]
  )

  const generateCSV = React.useCallback((items: Endpoint[]): string => {
    const bom = "\ufeff"
    const headers = [
      "url",
      "host",
      "location",
      "title",
      "status_code",
      "content_length",
      "content_type",
      "webserver",
      "tech",
      "response_body",
      "vhost",
      "created_at",
    ]

    const rows = items.map((item) =>
      [
        escapeCSV(item.url),
        escapeCSV(item.host),
        escapeCSV(item.location),
        escapeCSV(item.title),
        escapeCSV(item.statusCode),
        escapeCSV(item.contentLength),
        escapeCSV(item.contentType),
        escapeCSV(item.webserver),
        escapeCSV(formatArrayForCSV(item.tech)),
        escapeCSV(item.responseBody),
        escapeCSV(item.vhost),
        escapeCSV(formatDateForCSV(item.createdAt ?? "")),
      ].join(",")
    )

    return bom + [headers.join(","), ...rows].join("\n")
  }, [])

  const handleExportAll = React.useCallback(async () => {
    try {
      let blob: Blob | null = null

      if (scanId || targetId) {
        blob = await exportEndpoints()
      } else {
        const endpoints: Endpoint[] = data?.results || []
        if (endpoints.length === 0) {
          return
        }
        const csvContent = generateCSV(endpoints)
        blob = new Blob([csvContent], { type: "text/csv;charset=utf-8" })
      }

      if (!blob) return

      const prefix = scanId ? `scan-${scanId}` : targetId ? `target-${targetId}` : "endpoints"
      saveBlobAsFile(blob, `${prefix}-endpoints-${Date.now()}.csv`)
    } catch {
      toastFeedback.error(tToast("exportFailed"))
    }
  }, [data?.results, exportEndpoints, generateCSV, scanId, targetId, tToast])

  const handleExportSelected = React.useCallback(() => {
    if (selectedEndpoints.length === 0) {
      return
    }
    const csvContent = generateCSV(selectedEndpoints)
    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8" })
    const prefix = scanId ? `scan-${scanId}` : targetId ? `target-${targetId}` : "endpoints"
    saveBlobAsFile(blob, `${prefix}-endpoints-selected-${Date.now()}.csv`)
  }, [generateCSV, scanId, selectedEndpoints, targetId])

  const handleBulkDelete = React.useCallback(async () => {
    if (selectedEndpoints.length === 0) return

    setIsDeleting(true)
    try {
      const ids = selectedEndpoints.map((endpoint) => endpoint.id)
      await batchDeleteEndpoints.mutateAsync({ targetId: targetId!, ids })
      setSelectedEndpoints([])
      setBulkDeleteDialogOpen(false)
    } catch {
      void tToast
    } finally {
      setIsDeleting(false)
    }
  }, [batchDeleteEndpoints, selectedEndpoints, targetId, tToast])

  const cursorPaginationSummary: CursorPaginationSummary = {
    total: data?.totalSize ?? data?.total ?? data?.results?.length ?? 0,
  }
  const paginationNavigation = getCursorPaginationNavigation({
    currentPage: targetId ? activeTargetPage : activeScanPage,
    pageTokens: hasCursorScopeChanged ? { 1: undefined } : targetId ? pageTokens : scanPageTokens,
    nextPageToken: activeNextPageToken,
  })

  const activeFilters = activeBusinessQuery.filters ?? {}
  const statusCodeOptions = normalizeFilterOptions(statusCodeOptionsQuery.data)
  const techOptions = normalizeFilterOptions(techOptionsQuery.data)
  const webserverOptions = normalizeFilterOptions(webserverOptionsQuery.data)
  const contentTypeOptions = normalizeFilterOptions(contentTypeOptionsQuery.data)
  const vhostOptions = normalizeFilterOptions(vhostOptionsQuery.data)

  return {
    tCommon,
    tConfirm,
    targetId,
    isReadOnly,
    target,
    data,
    isLoading,
    error,
    refetch,
    isSearching: isFetching,
    filterQuery,
    commitFilterSearch,
    statusCodeFilter: activeFilters.statusCode ?? [],
    statusCodeOptions,
    techFilter: activeFilters.tech ?? [],
    techOptions,
    webserverFilter: activeFilters.webserver ?? [],
    webserverOptions,
    contentTypeFilter: activeFilters.contentType ?? [],
    contentTypeOptions,
    vhostFilter: activeFilters.vhost ?? [],
    vhostOptions,
    handleStatusCodeFilterChange: (values: string[]) => updateEndpointFacet("statusCode", values),
    handleTechFilterChange: (values: string[]) => updateEndpointFacet("tech", values),
    handleWebserverFilterChange: (values: string[]) => updateEndpointFacet("webserver", values),
    handleContentTypeFilterChange: (values: string[]) => updateEndpointFacet("contentType", values),
    handleVhostFilterChange: (values: string[]) => updateEndpointFacet("vhost", values.filter((value) => value === "true" || value === "false").slice(-1)),
    pagination,
    handlePaginationChange,
    cursorPaginationSummary,
    paginationNavigation,
    sortingMode: "server" as const,
    sorting,
    handleSortingChange,
    formatDate,
    columnVisibility,
    setColumnVisibility,
    endpointColumns,
    selectedEndpoints,
    handleSelectionChange,
    handleExportAll,
    handleExportSelected,
    handleBulkDelete,
    bulkAddDialogOpen,
    setBulkAddDialogOpen,
    bulkDeleteDialogOpen,
    setBulkDeleteDialogOpen,
    deleteDialogOpen,
    setDeleteDialogOpen,
    confirmDelete,
    deleteEndpoint,
    isDeleting,
  }
}

export type EndpointsDetailViewState = ReturnType<typeof useEndpointsDetailViewState>
