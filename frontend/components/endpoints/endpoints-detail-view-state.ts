import React from "react"
import { useLocale, useTranslations } from "next-intl"
import { toast } from "sonner"

import { useTargetEndpoints, useTarget } from "@/hooks/use-targets"
import { useDeleteEndpoint, useScanEndpoints } from "@/hooks/use-endpoints"
import { useSearchState } from "@/hooks/_shared/use-search-state"
import { buildPaginationInfo } from "@/hooks/_shared/pagination"
import { EndpointService } from "@/services/endpoint.service"
import { getDateLocale } from "@/lib/date-utils"
import { escapeCSV, formatArrayForCSV, formatDateForCSV } from "@/lib/csv-utils"
import { downloadBlob } from "@/lib/download-utils"
import { createEndpointColumns } from "./endpoints-columns"

import type { Endpoint } from "@/types/endpoint.types"

interface EndpointsDetailViewStateOptions {
  targetId?: number
  scanId?: number
}

export function useEndpointsDetailViewState({ targetId, scanId }: EndpointsDetailViewStateOptions) {
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [endpointToDelete, setEndpointToDelete] = React.useState<Endpoint | null>(null)
  const [selectedEndpoints, setSelectedEndpoints] = React.useState<Endpoint[]>([])
  const [bulkAddDialogOpen, setBulkAddDialogOpen] = React.useState(false)
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = React.useState(false)
  const [isDeleting, setIsDeleting] = React.useState(false)
  const [pagination, setPagination] = React.useState({
    pageIndex: 0,
    pageSize: 10,
  })
  const [filterQuery, setFilterQuery] = React.useState("")

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
        responseTime: tColumns("endpoint.responseTime"),
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

  const targetEndpointsQuery = useTargetEndpoints(
    targetId || 0,
    {
      page: pagination.pageIndex + 1,
      pageSize: pagination.pageSize,
      filter: filterQuery || undefined,
    },
    { enabled: !!targetId }
  )

  const scanEndpointsQuery = useScanEndpoints(
    scanId || 0,
    {
      page: pagination.pageIndex + 1,
      pageSize: pagination.pageSize,
    },
    { enabled: !!scanId },
    filterQuery || undefined
  )

  const activeQuery = targetId ? targetEndpointsQuery : scanEndpointsQuery
  const { data, isLoading, isFetching, error, refetch } = activeQuery

  const { isSearching, handleSearchChange: handleFilterChange } = useSearchState({
    isFetching,
    setSearchValue: setFilterQuery,
    onResetPage: () => setPagination((prev) => ({ ...prev, pageIndex: 0 })),
  })

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
      setPagination(newPagination)
    },
    []
  )

  const handleSelectionChange = React.useCallback((selectedRows: Endpoint[]) => {
    setSelectedEndpoints(selectedRows)
  }, [])

  const endpointColumns = React.useMemo(
    () =>
      createEndpointColumns({
        formatDate,
        t: translations,
      }),
    [formatDate, translations]
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

  const handleDownloadAll = React.useCallback(async () => {
    try {
      let blob: Blob | null = null

      if (scanId) {
        blob = await EndpointService.exportEndpointsByScanId(scanId)
      } else if (targetId) {
        blob = await EndpointService.exportEndpointsByTargetId(targetId)
      } else {
        const endpoints: Endpoint[] = data?.endpoints || []
        if (endpoints.length === 0) {
          return
        }
        const csvContent = generateCSV(endpoints)
        blob = new Blob([csvContent], { type: "text/csv;charset=utf-8" })
      }

      if (!blob) return

      const prefix = scanId ? `scan-${scanId}` : targetId ? `target-${targetId}` : "endpoints"
      downloadBlob(blob, `${prefix}-endpoints-${Date.now()}.csv`)
    } catch {
      toast.error(tToast("downloadFailed"))
    }
  }, [data?.endpoints, generateCSV, scanId, targetId, tToast])

  const handleDownloadSelected = React.useCallback(() => {
    if (selectedEndpoints.length === 0) {
      return
    }
    const csvContent = generateCSV(selectedEndpoints)
    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8" })
    const prefix = scanId ? `scan-${scanId}` : targetId ? `target-${targetId}` : "endpoints"
    downloadBlob(blob, `${prefix}-endpoints-selected-${Date.now()}.csv`)
  }, [generateCSV, scanId, selectedEndpoints, targetId])

  const handleBulkDelete = React.useCallback(async () => {
    if (selectedEndpoints.length === 0) return

    setIsDeleting(true)
    try {
      const ids = selectedEndpoints.map((endpoint) => endpoint.id)
      const result = await EndpointService.bulkDelete(ids)
      toast.success(tToast("deleteSuccess", { count: result.deletedCount }))
      setSelectedEndpoints([])
      setBulkDeleteDialogOpen(false)
      refetch()
    } catch {
      toast.error(tToast("deleteFailed"))
    } finally {
      setIsDeleting(false)
    }
  }, [selectedEndpoints, tToast, refetch])

  const paginationInfo = buildPaginationInfo({
    total: data?.pagination?.total ?? data?.endpoints?.length ?? 0,
    page: data?.pagination?.page ?? pagination.pageIndex + 1,
    pageSize: data?.pagination?.pageSize ?? pagination.pageSize,
    totalPages: data?.pagination?.totalPages,
    minTotalPages: 1,
  })

  return {
    tCommon,
    tConfirm,
    targetId,
    target,
    data,
    isLoading,
    error,
    refetch,
    isSearching,
    filterQuery,
    handleFilterChange,
    pagination,
    handlePaginationChange,
    paginationInfo,
    endpointColumns,
    selectedEndpoints,
    handleSelectionChange,
    handleDownloadAll,
    handleDownloadSelected,
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
