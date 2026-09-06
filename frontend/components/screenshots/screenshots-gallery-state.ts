import React from "react"
import { useTranslations } from "next-intl"
import type { SortingState } from "@tanstack/react-table"

import {
  useBulkDeleteScreenshots,
  useScanScreenshotFilterOptions,
  useScreenshotImageUrlResolver,
  useScanScreenshots,
  useTargetScreenshotFilterOptions,
  useTargetScreenshots,
} from "@/hooks/use-screenshots"
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
  type BusinessListQuery,
  type BusinessListSortableFieldConfig,
  type BusinessListSorting,
} from "@/components/shared/data-table/business-list-query"
import { useCursorPaginationScopeChange } from "@/components/shared/data-table/use-cursor-pagination-scope"
import type { Screenshot, ScreenshotFilterOption, ScreenshotFilterOptionField } from "@/types/screenshot.types"

const PAGE_SIZE_OPTIONS = [12, 24, 48]

const SCREENSHOT_SORTABLE_FIELDS: Record<string, BusinessListSortableFieldConfig> = {
  statusCode: { orderBy: "statusCode", firstDirection: "asc" },
  createdAt: { orderBy: "createdAt", firstDirection: "desc" },
}

const SCREENSHOT_DEFAULT_SORTING: BusinessListSorting = { field: "createdAt", direction: "desc" }

const SCREENSHOT_FILTER_CONFIG: BusinessListFilterCompilerConfig = {
  search: { field: "url", operator: "==" },
  facets: {
    statusCode: { field: "statusCode" },
  },
}

interface ScreenshotsGalleryStateOptions {
  targetId?: number
  scanId?: number
}

function useScopedScreenshotFilterOptions(
  field: ScreenshotFilterOptionField,
  { targetId, scanId }: ScreenshotsGalleryStateOptions
) {
  const targetOptions = useTargetScreenshotFilterOptions(targetId || 0, field, { enabled: !!targetId })
  const scanOptions = useScanScreenshotFilterOptions(scanId || 0, field, { enabled: !targetId && !!scanId })
  return targetId ? targetOptions : scanOptions
}

function normalizeFilterOptions(data: { results?: ScreenshotFilterOption[] } | undefined) {
  return data?.results ?? []
}

function uniqueNonEmptyValues(values: string[] | undefined) {
  return Array.from(new Set((values ?? []).map((value) => value.trim()).filter(Boolean)))
}

export function useScreenshotsGalleryState({
  targetId,
  scanId,
}: ScreenshotsGalleryStateOptions) {
  const [query, setQuery] = React.useState<BusinessListQuery>(() => createBusinessListQuery({
    pageSize: 12,
    sorting: SCREENSHOT_DEFAULT_SORTING,
  }))
  const [pageTokens, setPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })
  const [scanQuery, setScanQuery] = React.useState<BusinessListQuery>(() => createBusinessListQuery({
    pageSize: 12,
    sorting: SCREENSHOT_DEFAULT_SORTING,
  }))
  const [scanPageTokens, setScanPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })
  const [selectedIds, setSelectedIds] = React.useState<Set<number>>(new Set())
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [isDeleting, setIsDeleting] = React.useState(false)
  const [lightboxOpen, setLightboxOpen] = React.useState(false)
  const [lightboxIndex, setLightboxIndex] = React.useState(0)
  const cursorScopeKey = targetId ? `target:${targetId}` : `scan:${scanId ?? ""}`
  const hasCursorScopeChanged = useCursorPaginationScopeChange(cursorScopeKey)

  const t = useTranslations("pages.screenshots")
  const tCommon = useTranslations("common")
  const bulkDeleteScreenshots = useBulkDeleteScreenshots()
  const resolveImageUrl = useScreenshotImageUrlResolver(scanId)

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

  const compiledFilter = compileBusinessListFilter({ search: query.search, filters: query.filters }, SCREENSHOT_FILTER_CONFIG)
  const compiledOrderBy = compileBusinessListOrderBy(query.sorting, SCREENSHOT_SORTABLE_FIELDS)
  const compiledScanFilter = compileBusinessListFilter({ search: scanQuery.search, filters: scanQuery.filters }, SCREENSHOT_FILTER_CONFIG)
  const compiledScanOrderBy = compileBusinessListOrderBy(scanQuery.sorting, SCREENSHOT_SORTABLE_FIELDS)

  const targetQuery = useTargetScreenshots(
    targetId || 0,
    {
      pageSize,
      pageToken: targetPageToken,
      filter: compiledFilter,
      orderBy: compiledOrderBy,
    },
    { enabled: !!targetId }
  )

  const scanScreenshotsQuery = useScanScreenshots(
    scanId || 0,
    {
      pageSize: scanPageSize,
      pageToken: scanPageToken,
      filter: compiledScanFilter,
      orderBy: compiledScanOrderBy,
    },
    { enabled: !!scanId }
  )

  const statusCodeOptionsQuery = useScopedScreenshotFilterOptions("statusCode", { targetId, scanId })
  const activeQuery = targetId ? targetQuery : scanScreenshotsQuery
  const { data, isLoading, isFetching, error, refetch } = activeQuery
  const targetNextPageToken = getCurrentCursorNextPageToken(
    targetQuery.data?.nextPageToken,
    targetQuery.isPlaceholderData || hasCursorScopeChanged,
  )
  const scanNextPageToken = getCurrentCursorNextPageToken(
    scanScreenshotsQuery.data?.nextPageToken,
    scanScreenshotsQuery.isPlaceholderData || hasCursorScopeChanged,
  )
  const activeNextPageToken = targetId ? targetNextPageToken : scanNextPageToken

  const screenshots: Screenshot[] = React.useMemo(() => data?.results || [], [data])
  const cursorPaginationSummary = {
    total: data?.totalSize ?? data?.total ?? screenshots.length,
  }
  const paginationNavigation = getCursorPaginationNavigation({
    currentPage: targetId ? activeTargetPage : activeScanPage,
    pageTokens: hasCursorScopeChanged ? { 1: undefined } : targetId ? pageTokens : scanPageTokens,
    nextPageToken: activeNextPageToken,
  })

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

    setSelectedIds(new Set())
    setDeleteDialogOpen(false)
    setLightboxOpen(false)
    setLightboxIndex(0)
    resetPaging()
    resetScanPaging()
    setQuery((current) => setBusinessListPage(current, { pageIndex: 1, pageToken: undefined }))
    setScanQuery((current) => setBusinessListPage(current, { pageIndex: 1, pageToken: undefined }))
  }, [hasCursorScopeChanged, resetPaging, resetScanPaging])

  const toggleSelect = React.useCallback((id: number) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }, [])

  const selectAll = React.useCallback(() => {
    setSelectedIds(new Set(screenshots.map((s) => s.id)))
  }, [screenshots])

  const clearSelection = React.useCallback(() => {
    setSelectedIds(new Set())
  }, [])

  const handleBulkDelete = async () => {
    if (selectedIds.size === 0) return
    setIsDeleting(true)
    try {
      await bulkDeleteScreenshots.mutateAsync({ targetId: targetId!, ids: Array.from(selectedIds) })
      setSelectedIds(new Set())
      setDeleteDialogOpen(false)
    } finally {
      setIsDeleting(false)
    }
  }

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

  const updateScreenshotFacet = React.useCallback((field: string, values: string[]) => {
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

  const handlePageSizeChange = (value: string) => {
    const newPageSize = parseInt(value, 10)
    if (targetId) {
      resetPaging()
      setQuery((current) => applyBusinessListControlChange(current, { pageSize: newPageSize }))
      return
    }
    resetScanPaging()
    setScanQuery((current) => applyBusinessListControlChange(current, { pageSize: newPageSize }))
  }

  const handlePaginationChange = React.useCallback((newPagination: { pageIndex: number; pageSize: number }) => {
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
  }, [activeScanPage, activeTargetPage, pageSize, pageTokens, resetPaging, resetScanPaging, scanNextPageToken, scanPageSize, scanPageTokens, targetId, targetNextPageToken])

  const handleSortingChange = React.useCallback((nextSorting: SortingState) => {
    const nextColumnId = nextSorting[0]?.id
    if (!targetId) {
      resetScanPaging()
      if (!nextColumnId) {
        setScanQuery((current) => applyBusinessListControlChange(current, { sorting: SCREENSHOT_DEFAULT_SORTING }))
        return
      }
      setScanQuery((current) => toggleBusinessListSorting(
        current,
        nextColumnId,
        SCREENSHOT_SORTABLE_FIELDS,
        SCREENSHOT_DEFAULT_SORTING
      ))
      return
    }

    resetPaging()
    if (!nextColumnId) {
      setQuery((current) => applyBusinessListControlChange(current, { sorting: SCREENSHOT_DEFAULT_SORTING }))
      return
    }
    setQuery((current) => toggleBusinessListSorting(
      current,
      nextColumnId,
      SCREENSHOT_SORTABLE_FIELDS,
      SCREENSHOT_DEFAULT_SORTING
    ))
  }, [resetPaging, resetScanPaging, targetId])

  const openLightbox = (index: number) => {
    setLightboxIndex(index)
    setLightboxOpen(true)
  }

  const nextImage = () => {
    setLightboxIndex((prev) => (prev + 1) % screenshots.length)
  }

  const prevImage = () => {
    setLightboxIndex((prev) => (prev - 1 + screenshots.length) % screenshots.length)
  }

  const getImageUrl = React.useCallback(
    (screenshot: Screenshot) => {
      return resolveImageUrl(screenshot.id)
    },
    [resolveImageUrl]
  )

  const activeFilters = activeBusinessQuery.filters ?? {}

  return {
    t,
    tCommon,
    targetId,
    isLoading,
    isSearching: isFetching,
    error,
    refetch,
    data,
    screenshots,
    filterQuery,
    commitFilterSearch,
    pagination,
    handlePaginationChange,
    cursorPaginationSummary,
    paginationNavigation,
    sorting,
    handleSortingChange,
    statusCodeFilter: activeFilters.statusCode ?? [],
    statusCodeOptions: normalizeFilterOptions(statusCodeOptionsQuery.data),
    handleStatusCodeFilterChange: (values: string[]) => updateScreenshotFacet("statusCode", values),
    selectedIds,
    clearSelection,
    deleteDialogOpen,
    setDeleteDialogOpen,
    isDeleting,
    lightboxOpen,
    setLightboxOpen,
    lightboxIndex,
    pageSizeOptions: PAGE_SIZE_OPTIONS,
    toggleSelect,
    selectAll,
    handleBulkDelete,
    handlePageSizeChange,
    openLightbox,
    nextImage,
    prevImage,
    getImageUrl,
  }
}

export type ScreenshotsGalleryState = ReturnType<typeof useScreenshotsGalleryState>
