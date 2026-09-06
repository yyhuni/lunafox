import React from "react"
import { useTranslations } from "next-intl"
import type { SortingState } from "@tanstack/react-table"
import { toastFeedback } from "@/lib/toast-helpers"

import {
  useBulkDeleteDirectories,
  useExportDirectories,
  useTargetDirectories,
  useScanDirectories,
  useTargetDirectoryFilterOptions,
  useScanDirectoryFilterOptions,
} from "@/hooks/use-directories"
import { useTarget } from "@/hooks/use-targets"
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
import { saveBlobAsFile } from "@/lib/file-save-utils"
import { useDirectoryTableColumns } from "./directories-columns"
import { buildDirectoryCSV } from "./directory-csv"
import { composeWebsiteScopeFilter } from "@/lib/website-scope"

import type { Directory, DirectoryFilterOption, DirectoryFilterOptionField } from "@/types/directory.types"
import type { WebsiteAssetScope } from "@/types/website.types"
import type { CursorPaginationSummary } from "@/types/data-table.types"

const DIRECTORY_SORTABLE_FIELDS: Record<string, BusinessListSortableFieldConfig> = {
  status: { orderBy: "status", firstDirection: "asc" },
  contentLength: { orderBy: "contentLength", firstDirection: "desc" },
  createdAt: { orderBy: "createdAt", firstDirection: "desc" },
}

const DIRECTORY_DEFAULT_SORTING: BusinessListSorting = { field: "createdAt", direction: "desc" }

const DIRECTORY_FILTER_CONFIG: BusinessListFilterCompilerConfig = {
  search: { field: "url", operator: "==" },
  facets: {
    status: { field: "status" },
    contentType: { field: "contentType" },
  },
}

interface DirectoriesViewStateOptions {
  targetId?: number
  scanId?: number
  websiteScope?: WebsiteAssetScope
}

function useScopedDirectoryFilterOptions(
  field: DirectoryFilterOptionField,
  { targetId, scanId }: DirectoriesViewStateOptions
) {
  const targetOptions = useTargetDirectoryFilterOptions(targetId || 0, field, { enabled: !!targetId })
  const scanOptions = useScanDirectoryFilterOptions(scanId || 0, field, { enabled: !targetId && !!scanId })
  return targetId ? targetOptions : scanOptions
}

function normalizeFilterOptions(data: { results?: DirectoryFilterOption[] } | undefined) {
  return data?.results ?? []
}

function uniqueNonEmptyValues(values: string[] | undefined) {
  return Array.from(new Set((values ?? []).map((value) => value.trim()).filter(Boolean)))
}

export function useDirectoriesViewState({ targetId, scanId, websiteScope }: DirectoriesViewStateOptions) {
  const [selectedDirectories, setSelectedDirectories] = React.useState<Directory[]>([])
  const [bulkAddDialogOpen, setBulkAddDialogOpen] = React.useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [isDeleting, setIsDeleting] = React.useState(false)
  const [query, setQuery] = React.useState(() => createBusinessListQuery({
    pageSize: 10,
    sorting: DIRECTORY_DEFAULT_SORTING,
  }))
  const [pageTokens, setPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })
  const [scanQuery, setScanQuery] = React.useState(() => createBusinessListQuery({
    pageSize: 10,
    sorting: DIRECTORY_DEFAULT_SORTING,
  }))
  const [scanPageTokens, setScanPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })

  const tCommon = useTranslations("common")
  const tToast = useTranslations("toast")
  const tStatus = useTranslations("common.status")
  const isReadOnly = websiteScope?.readOnly === true
  const websiteScopeKey = websiteScope ? `${websiteScope.host}\u0000${websiteScope.url}` : ""
  const cursorScopeKey = targetId
    ? `target:${targetId}\u0000${websiteScopeKey}`
    : `scan:${scanId ?? ""}`
  const hasCursorScopeChanged = useCursorPaginationScopeChange(cursorScopeKey)
  const { columns } = useDirectoryTableColumns({ includeSelection: !isReadOnly })

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

  const compiledFilter = compileBusinessListFilter({ search: query.search, filters: query.filters }, DIRECTORY_FILTER_CONFIG)
  const scopedTargetFilter = websiteScope
    ? composeWebsiteScopeFilter(websiteScope, "websiteUrl", compiledFilter)
    : compiledFilter
  const compiledOrderBy = compileBusinessListOrderBy(query.sorting, DIRECTORY_SORTABLE_FIELDS)
  const compiledScanFilter = compileBusinessListFilter({ search: scanQuery.search, filters: scanQuery.filters }, DIRECTORY_FILTER_CONFIG)
  const compiledScanOrderBy = compileBusinessListOrderBy(scanQuery.sorting, DIRECTORY_SORTABLE_FIELDS)

  const { data: target } = useTarget(targetId || 0, { enabled: !!targetId })
  const bulkDeleteDirectories = useBulkDeleteDirectories()
  const exportDirectories = useExportDirectories({ targetId, scanId })

  const resetPaging = React.useCallback(() => {
    setPageTokens({ 1: undefined })
  }, [])
  const resetScanPaging = React.useCallback(() => {
    setScanPageTokens({ 1: undefined })
  }, [])

  React.useEffect(() => {
    if (!hasCursorScopeChanged) return

    setSelectedDirectories([])
    setBulkAddDialogOpen(false)
    setDeleteDialogOpen(false)
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

  const updateDirectoryFacet = React.useCallback((field: string, values: string[]) => {
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

  const targetQuery = useTargetDirectories(
    targetId || 0,
    {
      pageSize,
      pageToken: targetPageToken,
      filter: scopedTargetFilter,
      orderBy: compiledOrderBy,
    },
    { enabled: !!targetId, websiteScope }
  )

  const scanDirectoriesQuery = useScanDirectories(
    scanId || 0,
    {
      pageSize: scanPageSize,
      pageToken: scanPageToken,
      filter: compiledScanFilter,
      orderBy: compiledScanOrderBy,
    },
    { enabled: !!scanId }
  )

  const statusOptionsQuery = useScopedDirectoryFilterOptions("status", { targetId, scanId })
  const contentTypeOptionsQuery = useScopedDirectoryFilterOptions("contentType", { targetId, scanId })

  const activeQuery = targetId ? targetQuery : scanDirectoriesQuery
  const { data, isLoading, isFetching, error, refetch } = activeQuery
  const targetNextPageToken = getCurrentCursorNextPageToken(
    targetQuery.data?.nextPageToken,
    targetQuery.isPlaceholderData || hasCursorScopeChanged,
  )
  const scanNextPageToken = getCurrentCursorNextPageToken(
    scanDirectoriesQuery.data?.nextPageToken,
    scanDirectoriesQuery.isPlaceholderData || hasCursorScopeChanged,
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
        setScanQuery((current) => applyBusinessListControlChange(current, { sorting: DIRECTORY_DEFAULT_SORTING }))
        return
      }
      setScanQuery((current) => toggleBusinessListSorting(
        current,
        nextColumnId,
        DIRECTORY_SORTABLE_FIELDS,
        DIRECTORY_DEFAULT_SORTING
      ))
      return
    }

    resetPaging()
    if (!nextColumnId) {
      setQuery((current) => applyBusinessListControlChange(current, { sorting: DIRECTORY_DEFAULT_SORTING }))
      return
    }
    setQuery((current) => toggleBusinessListSorting(
      current,
      nextColumnId,
      DIRECTORY_SORTABLE_FIELDS,
      DIRECTORY_DEFAULT_SORTING
    ))
  }, [resetPaging, resetScanPaging, targetId])

  const directories: Directory[] = React.useMemo(() => {
    return data?.results ?? []
  }, [data])

  const cursorPaginationSummary: CursorPaginationSummary = {
    total: data?.totalSize ?? data?.total ?? directories.length,
  }
  const paginationNavigation = getCursorPaginationNavigation({
    currentPage: targetId ? activeTargetPage : activeScanPage,
    pageTokens: hasCursorScopeChanged ? { 1: undefined } : targetId ? pageTokens : scanPageTokens,
    nextPageToken: activeNextPageToken,
  })

  const activeFilters = activeBusinessQuery.filters ?? {}
  const statusOptions = normalizeFilterOptions(statusOptionsQuery.data)
  const contentTypeOptions = normalizeFilterOptions(contentTypeOptionsQuery.data)

  const handleSelectionChange = React.useCallback((selectedRows: Directory[]) => {
    setSelectedDirectories(selectedRows)
  }, [])

  const handleExportAll = React.useCallback(async () => {
    try {
      let blob: Blob | null = null

      if (scanId || targetId) {
        blob = await exportDirectories()
      } else if (directories.length > 0) {
        const csvContent = buildDirectoryCSV(directories)
        blob = new Blob([csvContent], { type: "text/csv;charset=utf-8" })
      }

      if (!blob) return

      const prefix = scanId ? `scan-${scanId}` : targetId ? `target-${targetId}` : "directories"
      saveBlobAsFile(blob, `${prefix}-directories-${Date.now()}.csv`)
    } catch {
      toastFeedback.error(tToast("exportFailed"))
    }
  }, [directories, exportDirectories, scanId, targetId, tToast])

  const handleExportSelected = React.useCallback(() => {
    if (selectedDirectories.length === 0) {
      return
    }
    const csvContent = buildDirectoryCSV(selectedDirectories)
    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8" })
    const prefix = scanId ? `scan-${scanId}` : targetId ? `target-${targetId}` : "directories"
    saveBlobAsFile(blob, `${prefix}-directories-selected-${Date.now()}.csv`)
  }, [scanId, selectedDirectories, targetId])

  const handleBulkDelete = React.useCallback(async () => {
    if (selectedDirectories.length === 0) return

    setIsDeleting(true)
    try {
      const ids = selectedDirectories.map((directory) => directory.id)
      await bulkDeleteDirectories.mutateAsync({ targetId: targetId!, ids })
      setSelectedDirectories([])
      setDeleteDialogOpen(false)
    } catch {
      void tToast
    } finally {
      setIsDeleting(false)
    }
  }, [bulkDeleteDirectories, selectedDirectories, targetId, tToast])

  return {
    tCommon,
    tStatus,
    targetId,
    isReadOnly,
    target,
    data,
    error,
    isLoading,
    isSearching: isFetching,
    refetch,
    columns,
    directories,
    filterQuery,
    commitFilterSearch,
    statusFilter: activeFilters.status ?? [],
    statusOptions,
    contentTypeFilter: activeFilters.contentType ?? [],
    contentTypeOptions,
    handleStatusFilterChange: (values: string[]) => updateDirectoryFacet("status", values),
    handleContentTypeFilterChange: (values: string[]) => updateDirectoryFacet("contentType", values),
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
    bulkAddDialogOpen,
    setBulkAddDialogOpen,
    deleteDialogOpen,
    setDeleteDialogOpen,
    selectedDirectories,
    isDeleting,
  }
}

export type DirectoriesViewState = ReturnType<typeof useDirectoriesViewState>
