import React from "react"
import { useLocale, useTranslations } from "next-intl"
import { toast } from "sonner"

import { useTargetWebSites, useScanWebSites } from "@/hooks/use-websites"
import { useTarget } from "@/hooks/use-targets"
import { useSearchState } from "@/hooks/_shared/use-search-state"
import { buildPaginationInfo, normalizePagination } from "@/hooks/_shared/pagination"
import { WebsiteService } from "@/services/website.service"
import { getDateLocale } from "@/lib/date-utils"
import { escapeCSV, formatArrayForCSV, formatDateForCSV } from "@/lib/csv-utils"
import { downloadBlob } from "@/lib/download-utils"
import { createWebSiteColumns } from "./websites-columns"

import type { WebSite } from "@/types/website.types"

interface WebSitesViewStateOptions {
  targetId?: number
  scanId?: number
}

export function useWebSitesViewState({ targetId, scanId }: WebSitesViewStateOptions) {
  const [pagination, setPagination] = React.useState({
    pageIndex: 0,
    pageSize: 10,
  })
  const [selectedWebSites, setSelectedWebSites] = React.useState<WebSite[]>([])
  const [bulkAddDialogOpen, setBulkAddDialogOpen] = React.useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [isDeleting, setIsDeleting] = React.useState(false)
  const [filterQuery, setFilterQuery] = React.useState("")

  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const tToast = useTranslations("toast")
  const tStatus = useTranslations("common.status")
  const locale = useLocale()

  const translations = React.useMemo(
    () => ({
      columns: {
        url: tColumns("common.url"),
        host: tColumns("website.host"),
        title: tColumns("endpoint.title"),
        status: tColumns("website.statusCode"),
        technologies: tColumns("endpoint.technologies"),
        contentLength: tColumns("endpoint.contentLength"),
        location: tColumns("endpoint.location"),
        webServer: tColumns("endpoint.webServer"),
        contentType: tColumns("endpoint.contentType"),
        responseBody: tColumns("endpoint.responseBody"),
        vhost: tColumns("endpoint.vhost"),
        responseHeaders: tColumns("website.responseHeaders"),
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

  const targetQuery = useTargetWebSites(
    targetId || 0,
    {
      page: pagination.pageIndex + 1,
      pageSize: pagination.pageSize,
      filter: filterQuery || undefined,
    },
    { enabled: !!targetId }
  )

  const scanQuery = useScanWebSites(
    scanId || 0,
    {
      page: pagination.pageIndex + 1,
      pageSize: pagination.pageSize,
      filter: filterQuery || undefined,
    },
    { enabled: !!scanId }
  )

  const activeQuery = targetId ? targetQuery : scanQuery
  const { data, isLoading, isFetching, error, refetch } = activeQuery

  const { isSearching, handleSearchChange: handleFilterChange } = useSearchState({
    isFetching,
    setSearchValue: setFilterQuery,
    onResetPage: () => setPagination((prev) => ({ ...prev, pageIndex: 0 })),
  })

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
      createWebSiteColumns({
        formatDate,
        t: translations,
      }),
    [formatDate, translations]
  )

  const websites: WebSite[] = React.useMemo(() => {
    return data?.results ?? []
  }, [data])

  const paginationInfo = data
    ? buildPaginationInfo({
        ...normalizePagination(data, pagination.pageIndex + 1, pagination.pageSize),
        minTotalPages: 1,
      })
    : undefined

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

  const handleDownloadAll = React.useCallback(async () => {
    try {
      let blob: Blob | null = null

      if (scanId) {
        blob = await WebsiteService.exportWebsitesByScanId(scanId)
      } else if (targetId) {
        blob = await WebsiteService.exportWebsitesByTargetId(targetId)
      } else if (websites.length > 0) {
        const csvContent = generateCSV(websites)
        blob = new Blob([csvContent], { type: "text/csv;charset=utf-8" })
      }

      if (!blob) return

      const prefix = scanId ? `scan-${scanId}` : targetId ? `target-${targetId}` : "websites"
      downloadBlob(blob, `${prefix}-websites-${Date.now()}.csv`)
    } catch {
      toast.error(tToast("downloadFailed"))
    }
  }, [generateCSV, scanId, targetId, tToast, websites])

  const handleDownloadSelected = React.useCallback(() => {
    if (selectedWebSites.length === 0) {
      return
    }
    const csvContent = generateCSV(selectedWebSites)
    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8" })
    const prefix = scanId ? `scan-${scanId}` : targetId ? `target-${targetId}` : "websites"
    downloadBlob(blob, `${prefix}-websites-selected-${Date.now()}.csv`)
  }, [generateCSV, scanId, selectedWebSites, targetId])

  const handleBulkDelete = React.useCallback(async () => {
    if (selectedWebSites.length === 0) return

    setIsDeleting(true)
    try {
      const ids = selectedWebSites.map((webSite) => webSite.id)
      const result = await WebsiteService.bulkDelete(ids)
      toast.success(tToast("deleteSuccess", { count: result.deletedCount }))
      setSelectedWebSites([])
      setDeleteDialogOpen(false)
      refetch()
    } catch {
      toast.error(tToast("deleteFailed"))
    } finally {
      setIsDeleting(false)
    }
  }, [selectedWebSites, tToast, refetch])

  return {
    tCommon,
    tStatus,
    targetId,
    target,
    data,
    error,
    isLoading,
    isSearching,
    refetch,
    columns,
    websites,
    filterQuery,
    handleFilterChange,
    pagination,
    setPagination,
    paginationInfo,
    handleSelectionChange,
    handleDownloadAll,
    handleDownloadSelected,
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
