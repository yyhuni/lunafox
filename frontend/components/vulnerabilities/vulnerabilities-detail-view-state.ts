import React from "react"
import { useLocale, useTranslations } from "next-intl"
import type { SortingState } from "@tanstack/react-table"

import { createVulnerabilityColumns } from "./vulnerabilities-columns"
import {
  useAllVulnerabilities,
  useBulkDeleteVulnerabilities,
  useBulkMarkAsReviewed,
  useBulkMarkAsUnreviewed,
  useGlobalVulnerabilityFilterOptions,
  useScanVulnerabilities,
  useScanVulnerabilityFilterOptions,
  useTargetVulnerabilities,
  useTargetVulnerabilityFilterOptions,
  useTargetVulnerabilityStats,
  useVulnerabilityStats,
} from "@/hooks/use-vulnerabilities"
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
  type BusinessListQuery,
  type BusinessListFilterCompilerConfig,
  type BusinessListSortableFieldConfig,
  type BusinessListSorting,
} from "@/components/shared/data-table/business-list-query"
import { useCursorPaginationScopeChange } from "@/components/shared/data-table/use-cursor-pagination-scope"
import { getDateLocale } from "@/lib/date-utils"
import { composeWebsiteScopeFilter } from "@/lib/website-scope"
import type { Vulnerability, VulnerabilityFilterOption, VulnerabilityFilterOptionField, VulnerabilitySeverityCounts } from "@/types/vulnerability.types"
import type { WebsiteAssetScope } from "@/types/website.types"
import type { ReviewFilter, SeverityFilter } from "./vulnerabilities-data-table"
import type { CursorPaginationSummary } from "@/types/data-table.types"

interface UseVulnerabilitiesDetailViewStateOptions {
  scanId?: number
  targetId?: number
  websiteScope?: WebsiteAssetScope
}

const VULNERABILITY_SORTABLE_FIELDS: Record<string, BusinessListSortableFieldConfig> = {
  createdAt: { orderBy: "createdAt", firstDirection: "desc" },
}

const VULNERABILITY_DEFAULT_SORTING: BusinessListSorting = { field: "createdAt", direction: "desc" }

const VULNERABILITY_FILTER_CONFIG: BusinessListFilterCompilerConfig = {
  search: { field: "url", operator: "==" },
  facets: {
    severity: { field: "severity", operator: "==" },
    source: { field: "source", operator: "==" },
    vulnType: { field: "vulnType", operator: "==" },
    isReviewed: { field: "isReviewed", operator: "==" },
  },
}

function normalizeFilterOptions(data: { results?: VulnerabilityFilterOption[] } | undefined) {
  return data?.results ?? []
}

function uniqueNonEmptyValues(values: string[] | undefined) {
  return Array.from(new Set((values ?? []).map((value) => value.trim()).filter(Boolean)))
}

function useScopedVulnerabilityFilterOptions(
  field: VulnerabilityFilterOptionField,
  { targetId, scanId }: Pick<UseVulnerabilitiesDetailViewStateOptions, "targetId" | "scanId">
) {
  const globalOptions = useGlobalVulnerabilityFilterOptions(field, { enabled: !targetId && !scanId })
  const targetOptions = useTargetVulnerabilityFilterOptions(targetId || 0, field, { enabled: !!targetId && !scanId })
  const scanOptions = useScanVulnerabilityFilterOptions(scanId || 0, field, { enabled: !!scanId })

  if (scanId) return scanOptions
  if (targetId) return targetOptions
  return globalOptions
}

export function useVulnerabilitiesDetailViewState({
  scanId,
  targetId,
  websiteScope,
}: UseVulnerabilitiesDetailViewStateOptions) {
  const [selectedVulnerabilities, setSelectedVulnerabilities] = React.useState<Vulnerability[]>([])
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [vulnerabilityToDelete, setVulnerabilityToDelete] = React.useState<Vulnerability | null>(null)
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = React.useState(false)
  const [isLoading, setIsLoading] = React.useState(false)
  const [reviewFilter, setReviewFilter] = React.useState<ReviewFilter>("all")
  const [severityFilter, setSeverityFilter] = React.useState<SeverityFilter>([])
  const [sourceFilter, setSourceFilter] = React.useState<string[]>([])
  const [vulnTypeFilter, setVulnTypeFilter] = React.useState<string[]>([])
  const [query, setQuery] = React.useState(() => createBusinessListQuery({
    pageSize: 10,
    sorting: VULNERABILITY_DEFAULT_SORTING,
  }))
  const [targetQueryState, setTargetQueryState] = React.useState(() => createBusinessListQuery({
    pageSize: 10,
    sorting: VULNERABILITY_DEFAULT_SORTING,
  }))
  const [scanQueryState, setScanQueryState] = React.useState(() => createBusinessListQuery({
    pageSize: 10,
    sorting: VULNERABILITY_DEFAULT_SORTING,
  }))
  const [pageTokens, setPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })
  const [targetPageTokens, setTargetPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })
  const [scanPageTokens, setScanPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })

  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const tTooltips = useTranslations("tooltips")
  const tSeverity = useTranslations("severity")
  const tConfirm = useTranslations("common.confirm")
  const locale = useLocale()

  const translations = React.useMemo(
    () => ({
      columns: {
        status: tColumns("common.status"),
        severity: tColumns("vulnerability.severity"),
        source: tColumns("vulnerability.source"),
        vulnType: tColumns("vulnerability.vulnType"),
        url: tColumns("common.url"),
        createdAt: tColumns("common.createdAt"),
      },
      actions: {
        selectAll: tCommon("actions.selectAll"),
        selectRow: tCommon("actions.selectRow"),
      },
      tooltips: {
        vulnDetails: tTooltips("vulnDetails"),
        reviewed: tTooltips("reviewed"),
        pending: tTooltips("pending"),
      },
      severity: {
        critical: tSeverity("critical"),
        high: tSeverity("high"),
        medium: tSeverity("medium"),
        low: tSeverity("low"),
        info: tSeverity("info"),
      },
    }),
    [tColumns, tCommon, tTooltips, tSeverity]
  )

  const bulkMarkAsReviewed = useBulkMarkAsReviewed()
  const bulkMarkAsUnreviewed = useBulkMarkAsUnreviewed()
  const bulkDeleteVulnerabilities = useBulkDeleteVulnerabilities()
  const isReadOnly = websiteScope?.readOnly === true
  const websiteScopeKey = websiteScope ? `${websiteScope.host}\u0000${websiteScope.url}` : ""
  const cursorScopeKey = scanId
    ? `scan:${scanId}`
    : targetId
      ? `target:${targetId}\u0000${websiteScopeKey}`
      : "global"
  const hasCursorScopeChanged = useCursorPaginationScopeChange(cursorScopeKey)

  const supportsReviewTabs = !scanId && !isReadOnly
  const activeBusinessQuery = scanId ? scanQueryState : targetId ? targetQueryState : query
  const activePageTokens = scanId ? scanPageTokens : targetId ? targetPageTokens : pageTokens
  const page = activeBusinessQuery.pageIndex ?? 1
  const pageSize = activeBusinessQuery.pageSize
  const activePage = hasCursorScopeChanged ? 1 : page
  const activePageToken = hasCursorScopeChanged ? undefined : activeBusinessQuery.pageToken
  const filterQuery = activeBusinessQuery.search ?? ""
  const pagination = React.useMemo(() => ({
    pageIndex: activePage - 1,
    pageSize,
  }), [activePage, pageSize])
  const sorting = React.useMemo<SortingState>(() => {
    const currentSorting = activeBusinessQuery.sorting
    if (!currentSorting) return []
    return [{ id: currentSorting.field, desc: currentSorting.direction === "desc" }]
  }, [activeBusinessQuery.sorting])

  const setActiveQuery = React.useCallback((updater: (current: BusinessListQuery) => BusinessListQuery) => {
    if (scanId) {
      setScanQueryState(updater)
      return
    }
    if (targetId) {
      setTargetQueryState(updater)
      return
    }
    setQuery(updater)
  }, [scanId, targetId])

  const resetActivePaging = React.useCallback(() => {
    if (scanId) {
      setScanPageTokens({ 1: undefined })
      return
    }
    if (targetId) {
      setTargetPageTokens({ 1: undefined })
      return
    }
    setPageTokens({ 1: undefined })
  }, [scanId, targetId])

  React.useEffect(() => {
    if (!hasCursorScopeChanged) return

    setSelectedVulnerabilities([])
    setDeleteDialogOpen(false)
    setBulkDeleteDialogOpen(false)
    setVulnerabilityToDelete(null)
    resetActivePaging()
    setActiveQuery((current) => setBusinessListPage(current, { pageIndex: 1, pageToken: undefined }))
  }, [hasCursorScopeChanged, resetActivePaging, setActiveQuery])

  const commitFilterSearch = React.useCallback((value: string) => {
    const normalizedSearch = preserveRawURLSearchInput(value)
    if ((activeBusinessQuery.search ?? "") === (normalizedSearch ?? "")) return
    resetActivePaging()
    setActiveQuery((current) => applyBusinessListControlChange(current, {
      search: normalizedSearch,
    }))
  }, [activeBusinessQuery.search, resetActivePaging, setActiveQuery])

  const handleReviewFilterChange = React.useCallback((filter: ReviewFilter) => {
    if (!supportsReviewTabs) return
    setReviewFilter(filter)
    resetActivePaging()
    setActiveQuery((current) => applyBusinessListControlChange(current, {}))
  }, [resetActivePaging, setActiveQuery, supportsReviewTabs])

  const handleSeverityFilterChange = React.useCallback((filter: SeverityFilter) => {
    const nextValues = uniqueNonEmptyValues(filter) as SeverityFilter
    setSeverityFilter(nextValues)
    resetActivePaging()
    setActiveQuery((current) => applyBusinessListControlChange(current, {
      filters: { ...current.filters, severity: nextValues },
    }))
  }, [resetActivePaging, setActiveQuery])

  const handleSourceFilterChange = React.useCallback((filter: string[]) => {
    const nextValues = uniqueNonEmptyValues(filter)
    setSourceFilter(nextValues)
    resetActivePaging()
    setActiveQuery((current) => applyBusinessListControlChange(current, {
      filters: { ...current.filters, source: nextValues },
    }))
  }, [resetActivePaging, setActiveQuery])

  const handleVulnTypeFilterChange = React.useCallback((filter: string[]) => {
    const nextValues = uniqueNonEmptyValues(filter)
    setVulnTypeFilter(nextValues)
    resetActivePaging()
    setActiveQuery((current) => applyBusinessListControlChange(current, {
      filters: { ...current.filters, vulnType: nextValues },
    }))
  }, [resetActivePaging, setActiveQuery])

  const reviewFilters: Record<string, string[]> = supportsReviewTabs
    ? { isReviewed: reviewFilter === "reviewed" ? ["true"] : reviewFilter === "pending" ? ["false"] : [] }
    : {}

  const compiledFilter = compileBusinessListFilter({
    search: query.search,
    filters: { ...query.filters, ...(supportsReviewTabs && !targetId ? reviewFilters : {}) },
  }, VULNERABILITY_FILTER_CONFIG)
  const compiledOrderBy = compileBusinessListOrderBy(query.sorting, VULNERABILITY_SORTABLE_FIELDS)
  const compiledTargetFilter = compileBusinessListFilter({
    search: targetQueryState.search,
    filters: { ...targetQueryState.filters, ...(supportsReviewTabs && targetId ? reviewFilters : {}) },
  }, VULNERABILITY_FILTER_CONFIG)
  const scopedTargetFilter = websiteScope
    ? composeWebsiteScopeFilter(websiteScope, "websiteUrl", compiledTargetFilter)
    : compiledTargetFilter
  const compiledTargetOrderBy = compileBusinessListOrderBy(targetQueryState.sorting, VULNERABILITY_SORTABLE_FIELDS)
  const compiledScanFilter = compileBusinessListFilter({
    search: scanQueryState.search,
    filters: scanQueryState.filters,
  }, VULNERABILITY_FILTER_CONFIG)
  const compiledScanOrderBy = compileBusinessListOrderBy(scanQueryState.sorting, VULNERABILITY_SORTABLE_FIELDS)

  const scanQuery = useScanVulnerabilities(
    scanId ?? 0,
    {
      pageSize: scanQueryState.pageSize,
      pageToken: scanId ? activePageToken : scanQueryState.pageToken,
      filter: compiledScanFilter,
      orderBy: compiledScanOrderBy,
    },
    { enabled: !!scanId }
  )
  const targetQuery = useTargetVulnerabilities(
    targetId ?? 0,
    {
      pageSize: targetQueryState.pageSize,
      pageToken: targetId && !scanId ? activePageToken : targetQueryState.pageToken,
      filter: scopedTargetFilter,
      orderBy: compiledTargetOrderBy,
    },
    { enabled: !!targetId && !scanId, websiteScope }
  )
  const allQuery = useAllVulnerabilities(
    {
      pageSize: query.pageSize,
      pageToken: !scanId && !targetId ? activePageToken : query.pageToken,
      filter: compiledFilter,
      orderBy: compiledOrderBy,
    },
    { enabled: !scanId && !targetId }
  )

  const sourceOptionsQuery = useScopedVulnerabilityFilterOptions("source", { targetId, scanId })
  const vulnTypeOptionsQuery = useScopedVulnerabilityFilterOptions("vulnType", { targetId, scanId })

  const activeQuery = scanId ? scanQuery : targetId ? targetQuery : allQuery
  const isQueryLoading = activeQuery.isLoading
  const allNextPageToken = getCurrentCursorNextPageToken(
    allQuery.data?.nextPageToken,
    allQuery.isPlaceholderData || hasCursorScopeChanged,
  )
  const targetNextPageToken = getCurrentCursorNextPageToken(
    targetQuery.data?.nextPageToken,
    targetQuery.isPlaceholderData || hasCursorScopeChanged,
  )
  const scanNextPageToken = getCurrentCursorNextPageToken(
    scanQuery.data?.nextPageToken,
    scanQuery.isPlaceholderData || hasCursorScopeChanged,
  )
  const activeNextPageToken = scanId
    ? scanNextPageToken
    : targetId
      ? targetNextPageToken
      : allNextPageToken

  const vulnerabilities = React.useMemo(
    () => activeQuery.data?.vulnerabilities ?? [],
    [activeQuery.data?.vulnerabilities]
  )

  React.useEffect(() => {
    if (allNextPageToken) {
      setPageTokens((tokens) => ({ ...tokens, [activePage + 1]: allNextPageToken }))
    }
  }, [activePage, allNextPageToken])

  React.useEffect(() => {
    if (targetNextPageToken) {
      setTargetPageTokens((tokens) => ({ ...tokens, [activePage + 1]: targetNextPageToken }))
    }
  }, [activePage, targetNextPageToken])

  React.useEffect(() => {
    if (scanNextPageToken) {
      setScanPageTokens((tokens) => ({ ...tokens, [activePage + 1]: scanNextPageToken }))
    }
  }, [activePage, scanNextPageToken])

  const cursorPaginationSummary: CursorPaginationSummary = {
    total: activeQuery.data?.totalSize ?? activeQuery.data?.total ?? vulnerabilities.length,
  }
  const paginationNavigation = getCursorPaginationNavigation({
    currentPage: activePage,
    pageTokens: hasCursorScopeChanged ? { 1: undefined } : activePageTokens,
    nextPageToken: activeNextPageToken,
  })

  const globalStatsQuery = useVulnerabilityStats({ enabled: !scanId && !targetId })
  const targetStatsQuery = useTargetVulnerabilityStats(targetId ?? 0, {
    enabled: !!targetId && !scanId && !isReadOnly,
  })

  const pendingCount = scanId
    ? 0
    : (targetId ? targetStatsQuery.data?.pendingCount : globalStatsQuery.data?.pendingCount) ?? 0
  const reviewedCount = scanId
    ? 0
    : (targetId ? targetStatsQuery.data?.reviewedCount : globalStatsQuery.data?.reviewedCount) ?? 0
  const statsData = isReadOnly
    ? undefined
    : targetId
      ? targetStatsQuery.data
      : globalStatsQuery.data
  const fallbackSeverityCounts = React.useMemo<VulnerabilitySeverityCounts>(() => ({
    critical: vulnerabilities.filter((item) => item.severity === "critical").length,
    high: vulnerabilities.filter((item) => item.severity === "high").length,
    medium: vulnerabilities.filter((item) => item.severity === "medium").length,
    low: vulnerabilities.filter((item) => item.severity === "low").length,
    info: vulnerabilities.filter((item) => item.severity === "info").length,
  }), [vulnerabilities])
  const severityCounts = React.useMemo<VulnerabilitySeverityCounts>(() => ({
    critical: statsData?.criticalCount ?? fallbackSeverityCounts.critical,
    high: statsData?.highCount ?? fallbackSeverityCounts.high,
    medium: statsData?.mediumCount ?? fallbackSeverityCounts.medium,
    low: statsData?.lowCount ?? fallbackSeverityCounts.low,
    info: statsData?.infoCount ?? fallbackSeverityCounts.info,
  }), [fallbackSeverityCounts, statsData])
  const totalCount = scanId ? cursorPaginationSummary.total : statsData?.total ?? cursorPaginationSummary.total
  const sourceOptions = normalizeFilterOptions(sourceOptionsQuery.data)
  const vulnTypeOptions = normalizeFilterOptions(vulnTypeOptionsQuery.data)

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

  const confirmDelete = React.useCallback(async () => {
    if (!vulnerabilityToDelete) return

    setDeleteDialogOpen(false)
    setIsLoading(true)
    setTimeout(() => {
      setVulnerabilityToDelete(null)
      setIsLoading(false)
    }, 1000)
  }, [vulnerabilityToDelete])

  const confirmBulkDelete = React.useCallback(async () => {
    if (selectedVulnerabilities.length === 0) return

    const selectionSnapshot = selectedVulnerabilities
    const ids = selectedVulnerabilities.map((vulnerability) => vulnerability.id)
    setBulkDeleteDialogOpen(false)
    setIsLoading(true)
    setSelectedVulnerabilities([])
    bulkDeleteVulnerabilities.mutate(ids, {
      onError: () => {
        setSelectedVulnerabilities((current) =>
          current.length === 0 ? selectionSnapshot : current
        )
      },
      onSettled: () => {
        setIsLoading(false)
      },
    })
  }, [bulkDeleteVulnerabilities, selectedVulnerabilities])

  const handleOpenBulkDeleteDialog = React.useCallback(() => {
    if (selectedVulnerabilities.length === 0) return
    setBulkDeleteDialogOpen(true)
  }, [selectedVulnerabilities.length])

  const handlePaginationChange = React.useCallback((newPagination: { pageIndex: number; pageSize: number }) => {
    const nextPage = newPagination.pageIndex + 1
    if (newPagination.pageSize !== pageSize) {
      resetActivePaging()
      setActiveQuery((current) => applyBusinessListControlChange(current, { pageSize: newPagination.pageSize }))
      return
    }

    if (nextPage === 1) {
      resetActivePaging()
      setActiveQuery((current) => setBusinessListPage(current, { pageIndex: 1, pageToken: undefined }))
      return
    }

    const transition = getCursorPageTransition({
      currentPage: activePage,
      pageTokens: activePageTokens,
      nextPageToken: activeNextPageToken,
      requestedPage: nextPage,
    })
    if (!transition.reachable) return

    setActiveQuery((current) => setBusinessListPage(current, {
      pageIndex: nextPage,
      pageToken: transition.pageToken,
    }))
  }, [activeNextPageToken, activePage, activePageTokens, pageSize, resetActivePaging, setActiveQuery])

  const handleSortingChange = React.useCallback((nextSorting: SortingState) => {
    const nextColumnId = nextSorting[0]?.id
    resetActivePaging()
    if (!nextColumnId) {
      setActiveQuery((current) => applyBusinessListControlChange(current, { sorting: VULNERABILITY_DEFAULT_SORTING }))
      return
    }
    setActiveQuery((current) => toggleBusinessListSorting(
      current,
      nextColumnId,
      VULNERABILITY_SORTABLE_FIELDS,
      VULNERABILITY_DEFAULT_SORTING
    ))
  }, [resetActivePaging, setActiveQuery])

  const handleBulkMarkAsReviewed = React.useCallback(() => {
    if (selectedVulnerabilities.length === 0) return
    const selectionSnapshot = selectedVulnerabilities
    const ids = selectedVulnerabilities.map((v) => v.id)
    setSelectedVulnerabilities([])
    bulkMarkAsReviewed.mutate(ids, {
      onError: () => {
        setSelectedVulnerabilities((current) =>
          current.length === 0 ? selectionSnapshot : current
        )
      },
    })
  }, [bulkMarkAsReviewed, selectedVulnerabilities])

  const handleBulkMarkAsPending = React.useCallback(() => {
    if (selectedVulnerabilities.length === 0) return
    const selectionSnapshot = selectedVulnerabilities
    const ids = selectedVulnerabilities.map((v) => v.id)
    setSelectedVulnerabilities([])
    bulkMarkAsUnreviewed.mutate(ids, {
      onError: () => {
        setSelectedVulnerabilities((current) =>
          current.length === 0 ? selectionSnapshot : current
        )
      },
    })
  }, [bulkMarkAsUnreviewed, selectedVulnerabilities])

  const vulnerabilityColumns = React.useMemo(
    () =>
      createVulnerabilityColumns({
        formatDate,
        t: translations,
        includeSelection: !isReadOnly,
      }),
    [formatDate, isReadOnly, translations]
  )

  return {
    tCommon,
    tConfirm,
    isReadOnly,
    activeQuery,
    isQueryLoading,
    isLoading,
    vulnerabilities,
    vulnerabilityColumns,
    filterQuery,
    commitFilterSearch,
    pagination,
    cursorPaginationSummary,
    paginationNavigation,
    handlePaginationChange,
    sorting,
    handleSortingChange,
    setSelectedVulnerabilities,
    selectedVulnerabilities,
    reviewFilter,
    handleReviewFilterChange,
    onReviewFilterChange: supportsReviewTabs ? handleReviewFilterChange : undefined,
    supportsReviewTabs,
    totalCount,
    pendingCount,
    reviewedCount,
    severityCounts,
    severityFilter,
    handleSeverityFilterChange,
    sourceFilter,
    handleSourceFilterChange,
    sourceOptions,
    vulnTypeFilter,
    handleVulnTypeFilterChange,
    vulnTypeOptions,
    handleBulkMarkAsReviewed,
    handleBulkMarkAsPending,
    handleOpenBulkDeleteDialog,
    deleteDialogOpen,
    setDeleteDialogOpen,
    vulnerabilityToDelete,
    confirmDelete,
    bulkDeleteDialogOpen,
    setBulkDeleteDialogOpen,
    confirmBulkDelete,
  }
}

export type VulnerabilitiesDetailViewState = ReturnType<typeof useVulnerabilitiesDetailViewState>
