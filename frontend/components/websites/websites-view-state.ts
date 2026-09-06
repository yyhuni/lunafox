import React from "react"
import { useTranslations } from "next-intl"
import type { SortingState, VisibilityState } from "@tanstack/react-table"
import { toastFeedback } from "@/lib/toast-helpers"

import {
  useBulkDeleteWebSites,
  useExportWebsites,
  useTargetWebSites,
  useScanWebSites,
  useTargetWebsiteFilterOptions,
  useScanWebsiteFilterOptions,
} from "@/hooks/use-websites"
import { useTarget } from "@/hooks/use-targets"
import {
  applyBusinessListControlChange,
  compileBusinessListOrderBy,
  createBusinessListQuery,
  getCurrentCursorNextPageToken,
  getCursorPageTransition,
  getCursorPaginationNavigation,
  preserveRawURLSearchInput,
  setBusinessListPage,
  toggleBusinessListSorting,
  type BusinessListQuery,
  type BusinessListSortableFieldConfig,
  type BusinessListSorting,
} from "@/components/shared/data-table/business-list-query"
import { useCursorPaginationScopeChange } from "@/components/shared/data-table/use-cursor-pagination-scope"
import { escapeCSV, formatArrayForCSV, formatDateForCSV } from "@/lib/csv-utils"
import { saveBlobAsFile } from "@/lib/file-save-utils"
import { useWebSiteTableColumns } from "./websites-columns"

import type { FilterOption, WebSite, WebsiteFilterOptionField } from "@/types/website.types"
import type { CursorPaginationSummary } from "@/types/data-table.types"

function useScopedWebsiteFilterOptions(
  field: WebsiteFilterOptionField,
  { targetId, scanId }: WebSitesViewStateOptions
) {
  const targetOptions = useTargetWebsiteFilterOptions(targetId || 0, field, { enabled: !!targetId })
  const scanOptions = useScanWebsiteFilterOptions(scanId || 0, field, { enabled: !targetId && !!scanId })
  return targetId ? targetOptions : scanOptions
}

function normalizeFilterOptions(data: { results?: FilterOption[] } | undefined) {
  return data?.results ?? []
}

const WEBSITE_SORTABLE_FIELDS: Record<string, BusinessListSortableFieldConfig> = {
  statusCode: { orderBy: "statusCode", firstDirection: "asc" },
  contentLength: { orderBy: "contentLength", firstDirection: "desc" },
  createdAt: { orderBy: "createdAt", firstDirection: "desc" },
}

const WEBSITE_DEFAULT_SORTING: BusinessListSorting = { field: "createdAt", direction: "desc" }
export const DEFAULT_WEBSITE_COLUMN_VISIBILITY: VisibilityState = {
  host: false,
  location: false,
  responseBody: false,
  responseHeaders: false,
}

function quoteFilterValue(value: string) {
  return value.replace(/\\/g, "\\\\").replace(/"/g, "\\\"")
}

function uniqueNonEmptyValues(values: string[] | undefined) {
  return Array.from(new Set((values ?? []).map((value) => value.trim()).filter(Boolean)))
}

function compileMultiValueFilter(field: string, values: string[]) {
  const predicates = uniqueNonEmptyValues(values).map((value) => `${field}="${quoteFilterValue(value)}"`)
  if (predicates.length === 0) return undefined
  return predicates.length === 1 ? predicates[0] : `(${predicates.join(" || ")})`
}

function compileWebsiteFilter(query: Pick<BusinessListQuery, "search" | "filters">) {
  const clauses: string[] = []
  const search = preserveRawURLSearchInput(query.search)
  if (search) {
    clauses.push(`url=="${quoteFilterValue(search)}"`)
  }

  for (const [field, values] of [
    ["statusCode", query.filters?.statusCode],
    ["tech", query.filters?.tech],
    ["webserver", query.filters?.webserver],
    ["contentType", query.filters?.contentType],
  ] as const) {
    const clause = compileMultiValueFilter(field, values)
    if (clause) clauses.push(clause)
  }

  const vhost = uniqueNonEmptyValues(query.filters?.vhost).filter((value) => value === "true" || value === "false")
  if (vhost.length === 1) {
    clauses.push(`vhost="${vhost[0]}"`)
  }

  return clauses.length > 0 ? clauses.join(" && ") : undefined
}

type WebsiteListRestoreState = {
  query: BusinessListQuery
  pageTokens: Record<number, string | undefined>
}

interface WebSitesViewStateOptions {
  targetId?: number
  scanId?: number
  initialTargetState?: WebsiteListRestoreState
}

export function useWebSitesViewState({ targetId, scanId, initialTargetState }: WebSitesViewStateOptions) {
  const [selectedWebSites, setSelectedWebSites] = React.useState<WebSite[]>([])
  const [columnVisibility, setColumnVisibility] = React.useState<VisibilityState>(
    DEFAULT_WEBSITE_COLUMN_VISIBILITY
  )
  const [bulkAddDialogOpen, setBulkAddDialogOpen] = React.useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [isDeleting, setIsDeleting] = React.useState(false)
  const [query, setQuery] = React.useState(() => initialTargetState?.query ?? createBusinessListQuery({
    pageSize: 10,
    sorting: WEBSITE_DEFAULT_SORTING,
  }))
  const [pageTokens, setPageTokens] = React.useState<Record<number, string | undefined>>(
    () => initialTargetState?.pageTokens ?? { 1: undefined }
  )
  const [scanQuery, setScanQuery] = React.useState(() => createBusinessListQuery({
    pageSize: 10,
    sorting: WEBSITE_DEFAULT_SORTING,
  }))
  const [scanPageTokens, setScanPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })
  const cursorScopeKey = targetId ? `target:${targetId}` : `scan:${scanId ?? ""}`
  const hasCursorScopeChanged = useCursorPaginationScopeChange(cursorScopeKey)

  const tCommon = useTranslations("common")
  const tToast = useTranslations("toast")
  const tStatus = useTranslations("common.status")
  const { columns, formatDate } = useWebSiteTableColumns()

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

  const compiledFilter = compileWebsiteFilter({ search: query.search, filters: query.filters })
  const compiledOrderBy = compileBusinessListOrderBy(query.sorting, WEBSITE_SORTABLE_FIELDS)
  const compiledScanFilter = compileWebsiteFilter({ search: scanQuery.search, filters: scanQuery.filters })
  const compiledScanOrderBy = compileBusinessListOrderBy(scanQuery.sorting, WEBSITE_SORTABLE_FIELDS)

  const { data: target } = useTarget(targetId || 0, { enabled: !!targetId })
  const bulkDeleteWebsites = useBulkDeleteWebSites()
  const exportWebsites = useExportWebsites({ targetId, scanId })

  const resetPaging = React.useCallback(() => {
    setPageTokens({ 1: undefined })
  }, [])
  const resetScanPaging = React.useCallback(() => {
    setScanPageTokens({ 1: undefined })
  }, [])

  React.useEffect(() => {
    if (!hasCursorScopeChanged) return

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

  const updateWebsiteFacet = React.useCallback((field: string, values: string[]) => {
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

  const targetQuery = useTargetWebSites(
    targetId || 0,
    {
      pageSize,
      pageToken: targetPageToken,
      filter: compiledFilter,
      orderBy: compiledOrderBy,
    },
    { enabled: !!targetId }
  )

  const scanWebsitesQuery = useScanWebSites(
    scanId || 0,
    {
      pageSize: scanPageSize,
      pageToken: scanPageToken,
      filter: compiledScanFilter,
      orderBy: compiledScanOrderBy,
    },
    { enabled: !!scanId }
  )

  const statusCodeOptionsQuery = useScopedWebsiteFilterOptions("statusCode", { targetId, scanId })
  const techOptionsQuery = useScopedWebsiteFilterOptions("tech", { targetId, scanId })
  const webserverOptionsQuery = useScopedWebsiteFilterOptions("webserver", { targetId, scanId })
  const contentTypeOptionsQuery = useScopedWebsiteFilterOptions("contentType", { targetId, scanId })
  const vhostOptionsQuery = useScopedWebsiteFilterOptions("vhost", { targetId, scanId })

  const activeQuery = targetId ? targetQuery : scanWebsitesQuery
  const { data, isLoading, isFetching, error, refetch } = activeQuery
  const targetNextPageToken = getCurrentCursorNextPageToken(
    targetQuery.data?.nextPageToken,
    targetQuery.isPlaceholderData || hasCursorScopeChanged,
  )
  const scanNextPageToken = getCurrentCursorNextPageToken(
    scanWebsitesQuery.data?.nextPageToken,
    scanWebsitesQuery.isPlaceholderData || hasCursorScopeChanged,
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
        setScanQuery((current) => applyBusinessListControlChange(current, { sorting: WEBSITE_DEFAULT_SORTING }))
        return
      }
      setScanQuery((current) => toggleBusinessListSorting(
        current,
        nextColumnId,
        WEBSITE_SORTABLE_FIELDS,
        WEBSITE_DEFAULT_SORTING
      ))
      return
    }

    resetPaging()
    if (!nextColumnId) {
      setQuery((current) => applyBusinessListControlChange(current, { sorting: WEBSITE_DEFAULT_SORTING }))
      return
    }
    setQuery((current) => toggleBusinessListSorting(
      current,
      nextColumnId,
      WEBSITE_SORTABLE_FIELDS,
      WEBSITE_DEFAULT_SORTING
    ))
  }, [resetPaging, resetScanPaging, targetId])

  const websites: WebSite[] = React.useMemo(() => {
    return data?.results ?? []
  }, [data])

  const cursorPaginationSummary: CursorPaginationSummary = {
    total: data?.totalSize ?? data?.total ?? websites.length,
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

  const handleSelectionChange = React.useCallback((selectedRows: WebSite[]) => {
    setSelectedWebSites(selectedRows)
  }, [])

  const generateCSV = React.useCallback(
    (items: WebSite[]): string => {
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
          escapeCSV(formatDateForCSV(item.createdAt)),
        ].join(",")
      )

      return bom + [headers.join(","), ...rows].join("\n")
    },
    []
  )

  const handleExportAll = React.useCallback(async () => {
    try {
      let blob: Blob | null = null

      if (scanId || targetId) {
        blob = await exportWebsites()
      } else if (websites.length > 0) {
        const csvContent = generateCSV(websites)
        blob = new Blob([csvContent], { type: "text/csv;charset=utf-8" })
      }

      if (!blob) return

      const prefix = scanId ? `scan-${scanId}` : targetId ? `target-${targetId}` : "websites"
      saveBlobAsFile(blob, `${prefix}-websites-${Date.now()}.csv`)
    } catch {
      toastFeedback.error(tToast("exportFailed"))
    }
  }, [exportWebsites, generateCSV, scanId, targetId, tToast, websites])

  const handleExportSelected = React.useCallback(() => {
    if (selectedWebSites.length === 0) {
      return
    }
    const csvContent = generateCSV(selectedWebSites)
    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8" })
    const prefix = scanId ? `scan-${scanId}` : targetId ? `target-${targetId}` : "websites"
    saveBlobAsFile(blob, `${prefix}-websites-selected-${Date.now()}.csv`)
  }, [generateCSV, scanId, selectedWebSites, targetId])

  const handleBulkDelete = React.useCallback(async () => {
    if (selectedWebSites.length === 0) return

    setIsDeleting(true)
    try {
      const ids = selectedWebSites.map((webSite) => webSite.id)
      await bulkDeleteWebsites.mutateAsync({ targetId: targetId!, ids })
      setSelectedWebSites([])
      setDeleteDialogOpen(false)
    } catch {
      void tToast
    } finally {
      setIsDeleting(false)
    }
  }, [bulkDeleteWebsites, selectedWebSites, targetId, tToast])

  return {
    tCommon,
    tStatus,
    targetId,
    target,
    query,
    pageTokens,
    data,
    error,
    isLoading,
    isSearching: isFetching,
    refetch,
    columns,
    websites,
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
    handleStatusCodeFilterChange: (values: string[]) => updateWebsiteFacet("statusCode", values),
    handleTechFilterChange: (values: string[]) => updateWebsiteFacet("tech", values),
    handleWebserverFilterChange: (values: string[]) => updateWebsiteFacet("webserver", values),
    handleContentTypeFilterChange: (values: string[]) => updateWebsiteFacet("contentType", values),
    handleVhostFilterChange: (values: string[]) => updateWebsiteFacet("vhost", values.filter((value) => value === "true" || value === "false").slice(-1)),
    pagination,
    cursorPaginationSummary,
    paginationNavigation,
    handlePaginationChange,
    sortingMode: "server" as const,
    sorting,
    handleSortingChange,
    formatDate,
    handleSelectionChange,
    columnVisibility,
    setColumnVisibility,
    handleExportAll,
    handleExportSelected,
    handleBulkDelete,
    bulkAddDialogOpen,
    setBulkAddDialogOpen,
    deleteDialogOpen,
    setDeleteDialogOpen,
    selectedWebSites,
    isDeleting,
  }
}

export type WebSitesViewState = ReturnType<typeof useWebSitesViewState>
