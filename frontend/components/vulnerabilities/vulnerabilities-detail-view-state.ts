import React from "react"
import { useLocale, useTranslations } from "next-intl"

import { createVulnerabilityColumns } from "./vulnerabilities-columns"
import {
  useAllVulnerabilities,
  useBulkMarkAsReviewed,
  useBulkMarkAsUnreviewed,
  useMarkAsReviewed,
  useMarkAsUnreviewed,
  useScanVulnerabilities,
  useTargetVulnerabilities,
  useTargetVulnerabilityStats,
  useVulnerabilityStats,
} from "@/hooks/use-vulnerabilities"
import { buildPaginationInfo, normalizePagination } from "@/hooks/_shared/pagination"
import { getDateLocale } from "@/lib/date-utils"

import type { Vulnerability } from "@/types/vulnerability.types"
import type { ReviewFilter, SeverityFilter } from "./vulnerabilities-data-table"

interface UseVulnerabilitiesDetailViewStateOptions {
  scanId?: number
  targetId?: number
}

export function useVulnerabilitiesDetailViewState({
  scanId,
  targetId,
}: UseVulnerabilitiesDetailViewStateOptions) {
  const [selectedVulnerabilities, setSelectedVulnerabilities] = React.useState<Vulnerability[]>([])
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [vulnerabilityToDelete, setVulnerabilityToDelete] = React.useState<Vulnerability | null>(null)
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = React.useState(false)
  const [isLoading, setIsLoading] = React.useState(false)
  const [detailDialogOpen, setDetailDialogOpen] = React.useState(false)
  const [selectedVulnerability, setSelectedVulnerability] = React.useState<Vulnerability | null>(null)
  const [reviewFilter, setReviewFilter] = React.useState<ReviewFilter>("all")
  const [severityFilter, setSeverityFilter] = React.useState<SeverityFilter>("all")
  const [pagination, setPagination] = React.useState({
    pageIndex: 0,
    pageSize: 10,
  })
  const [filterQuery, setFilterQuery] = React.useState("")

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
        details: tCommon("actions.details"),
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

  const markAsReviewed = useMarkAsReviewed()
  const markAsUnreviewed = useMarkAsUnreviewed()
  const bulkMarkAsReviewed = useBulkMarkAsReviewed()
  const bulkMarkAsUnreviewed = useBulkMarkAsUnreviewed()

  const handleFilterChange = React.useCallback((value: string) => {
    setFilterQuery(value)
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }, [])

  const handleReviewFilterChange = React.useCallback((filter: ReviewFilter) => {
    setReviewFilter(filter)
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }, [])

  const handleSeverityFilterChange = React.useCallback((filter: SeverityFilter) => {
    setSeverityFilter(filter)
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }, [])

  const isReviewedParam = reviewFilter === "all" ? undefined : reviewFilter === "reviewed"
  const severityParam = severityFilter === "all" ? undefined : severityFilter

  const paginationParams = {
    page: pagination.pageIndex + 1,
    pageSize: pagination.pageSize,
    isReviewed: isReviewedParam,
    severity: severityParam,
  }

  const scanQuery = useScanVulnerabilities(
    scanId ?? 0,
    paginationParams,
    { enabled: !!scanId },
    filterQuery || undefined
  )
  const targetQuery = useTargetVulnerabilities(
    targetId ?? 0,
    paginationParams,
    { enabled: !!targetId && !scanId },
    filterQuery || undefined
  )
  const allQuery = useAllVulnerabilities(
    paginationParams,
    { enabled: !scanId && !targetId },
    filterQuery || undefined
  )

  const activeQuery = scanId ? scanQuery : targetId ? targetQuery : allQuery
  const isQueryLoading = activeQuery.isLoading

  const vulnerabilities = activeQuery.data?.vulnerabilities ?? []
  const paginationInfo = buildPaginationInfo({
    ...(activeQuery.data?.pagination
      ? normalizePagination(activeQuery.data.pagination, pagination.pageIndex + 1, pagination.pageSize)
      : {
          total: vulnerabilities.length,
          page: pagination.pageIndex + 1,
          pageSize: pagination.pageSize,
        }),
    minTotalPages: 1,
  })

  const globalStatsQuery = useVulnerabilityStats({ enabled: !scanId && !targetId })
  const targetStatsQuery = useTargetVulnerabilityStats(targetId ?? 0, {
    enabled: !!targetId && !scanId,
  })

  const pendingCount = scanId
    ? 0
    : (targetId ? targetStatsQuery.data?.pendingCount : globalStatsQuery.data?.pendingCount) ?? 0
  const reviewedCount = scanId
    ? 0
    : (targetId ? targetStatsQuery.data?.reviewedCount : globalStatsQuery.data?.reviewedCount) ?? 0

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

  const handleViewDetail = React.useCallback((vulnerability: Vulnerability) => {
    setSelectedVulnerability(vulnerability)
    setDetailDialogOpen(true)
  }, [])

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

    setBulkDeleteDialogOpen(false)
    setIsLoading(true)
    setTimeout(() => {
      setSelectedVulnerabilities([])
      setIsLoading(false)
    }, 1000)
  }, [selectedVulnerabilities.length])

  const handlePaginationChange = React.useCallback((newPagination: { pageIndex: number; pageSize: number }) => {
    setPagination(newPagination)
  }, [])

  const handleToggleReview = React.useCallback(
    (vulnerability: Vulnerability) => {
      if (vulnerability.isReviewed) {
        markAsUnreviewed.mutate(vulnerability.id)
      } else {
        markAsReviewed.mutate(vulnerability.id)
      }
    },
    [markAsReviewed, markAsUnreviewed]
  )

  const handleBulkMarkAsReviewed = React.useCallback(() => {
    if (selectedVulnerabilities.length === 0) return
    const ids = selectedVulnerabilities.map((v) => v.id)
    bulkMarkAsReviewed.mutate(ids, {
      onSuccess: () => {
        setSelectedVulnerabilities([])
      },
    })
  }, [bulkMarkAsReviewed, selectedVulnerabilities])

  const handleBulkMarkAsPending = React.useCallback(() => {
    if (selectedVulnerabilities.length === 0) return
    const ids = selectedVulnerabilities.map((v) => v.id)
    bulkMarkAsUnreviewed.mutate(ids, {
      onSuccess: () => {
        setSelectedVulnerabilities([])
      },
    })
  }, [bulkMarkAsUnreviewed, selectedVulnerabilities])

  const vulnerabilityColumns = React.useMemo(
    () =>
      createVulnerabilityColumns({
        formatDate,
        handleViewDetail,
        onToggleReview: handleToggleReview,
        t: translations,
      }),
    [formatDate, handleViewDetail, handleToggleReview, translations]
  )

  return {
    tCommon,
    tConfirm,
    activeQuery,
    isQueryLoading,
    isLoading,
    vulnerabilities,
    vulnerabilityColumns,
    filterQuery,
    handleFilterChange,
    pagination,
    setPagination,
    paginationInfo,
    handlePaginationChange,
    setSelectedVulnerabilities,
    selectedVulnerabilities,
    reviewFilter,
    handleReviewFilterChange,
    pendingCount,
    reviewedCount,
    severityFilter,
    handleSeverityFilterChange,
    handleBulkMarkAsReviewed,
    handleBulkMarkAsPending,
    detailDialogOpen,
    setDetailDialogOpen,
    selectedVulnerability,
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
