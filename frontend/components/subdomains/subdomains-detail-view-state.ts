import React from "react"
import { useLocale, useTranslations } from "next-intl"
import { toast } from "sonner"

import { useTarget } from "@/hooks/use-targets"
import { useTargetSubdomains, useScanSubdomains } from "@/hooks/use-subdomains"
import { useSearchState } from "@/hooks/_shared/use-search-state"
import { buildPaginationInfo, normalizePagination } from "@/hooks/_shared/pagination"
import { SubdomainService } from "@/services/subdomain.service"
import { getDateLocale } from "@/lib/date-utils"
import { escapeCSV, formatDateForCSV } from "@/lib/csv-utils"
import { downloadBlob } from "@/lib/download-utils"
import { createSubdomainColumns } from "./subdomains-columns"

import type { Subdomain } from "@/types/subdomain.types"

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
  const [pagination, setPagination] = React.useState({
    pageIndex: 0,
    pageSize: 10,
  })
  const [filterQuery, setFilterQuery] = React.useState("")

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

  const targetSubdomainsQuery = useTargetSubdomains(
    targetId || 0,
    {
      page: pagination.pageIndex + 1,
      pageSize: pagination.pageSize,
      filter: filterQuery || undefined,
    },
    { enabled: !!targetId }
  )

  const scanSubdomainsQuery = useScanSubdomains(
    scanId || 0,
    {
      page: pagination.pageIndex + 1,
      pageSize: pagination.pageSize,
      filter: filterQuery || undefined,
    },
    { enabled: !!scanId }
  )

  const activeQuery = targetId ? targetSubdomainsQuery : scanSubdomainsQuery
  const { data: subdomainsData, isLoading, isFetching, error, refetch } = activeQuery
  const { isSearching, handleSearchChange: handleFilterChange } = useSearchState({
    isFetching,
    setSearchValue: setFilterQuery,
    onResetPage: () => setPagination((prev) => ({ ...prev, pageIndex: 0 })),
  })

  const paginationInfo = subdomainsData
    ? buildPaginationInfo({
        ...normalizePagination(subdomainsData, pagination.pageIndex + 1, pagination.pageSize),
        minTotalPages: 1,
      })
    : undefined

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

  const handlePaginationChange = React.useCallback(
    (newPagination: { pageIndex: number; pageSize: number }) => {
      setPagination(newPagination)
    },
    []
  )

  const generateCSV = React.useCallback((items: Subdomain[]): string => {
    const bom = "\ufeff"
    const headers = ["name", "created_at"]

    const rows = items.map((item) =>
      [escapeCSV(item.name), escapeCSV(formatDateForCSV(item.createdAt))].join(",")
    )

    return bom + [headers.join(","), ...rows].join("\n")
  }, [])

  const subdomains: Subdomain[] = React.useMemo(() => {
    if (!subdomainsData?.results) return []
    return subdomainsData.results.map((item) => ({
      id: item.id,
      name: item.name,
      createdAt: item.createdAt,
    }))
  }, [subdomainsData])

  const handleDownloadAll = React.useCallback(async () => {
    try {
      let blob: Blob | null = null

      if (scanId) {
        blob = await SubdomainService.exportSubdomainsByScanId(scanId)
      } else if (targetId) {
        blob = await SubdomainService.exportSubdomainsByTargetId(targetId)
      } else if (subdomains.length > 0) {
        const csvContent = generateCSV(subdomains)
        blob = new Blob([csvContent], { type: "text/csv;charset=utf-8" })
      }

      if (!blob) return

      const prefix = scanId ? `scan-${scanId}` : targetId ? `target-${targetId}` : "subdomains"
      downloadBlob(blob, `${prefix}-subdomains-${Date.now()}.csv`)
    } catch (error) {
      void error
    }
  }, [generateCSV, scanId, subdomains, targetId])

  const handleDownloadSelected = React.useCallback(() => {
    if (selectedSubdomains.length === 0) {
      return
    }
    const csvContent = generateCSV(selectedSubdomains)
    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8" })
    downloadBlob(blob, `subdomains-selected-${scanId ?? targetId ?? "all"}-${Date.now()}.csv`)
  }, [generateCSV, scanId, selectedSubdomains, targetId])

  const handleBulkDelete = React.useCallback(async () => {
    if (selectedSubdomains.length === 0) return

    setIsDeleting(true)
    try {
      const ids = selectedSubdomains.map((subdomain) => subdomain.id)
      const result = await SubdomainService.bulkDeleteSubdomains(ids)
      toast.success(tToast("deleteSuccess", { count: result.deletedCount }))
      setSelectedSubdomains([])
      setDeleteDialogOpen(false)
      refetch()
    } catch {
      toast.error(tToast("deleteFailed"))
    } finally {
      setIsDeleting(false)
    }
  }, [selectedSubdomains, tToast, refetch])

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
    filterQuery,
    handleFilterChange,
    isSearching,
    pagination,
    setPagination,
    paginationInfo,
    handlePaginationChange,
    handleDownloadAll,
    handleDownloadSelected,
    handleBulkDelete,
  }
}

export type SubdomainsDetailViewState = ReturnType<typeof useSubdomainsDetailViewState>
