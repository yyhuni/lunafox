import React from "react"
import { useLocale, useTranslations } from "next-intl"
import { toast } from "sonner"

import { useTargetIPAddresses, useScanIPAddresses } from "@/hooks/use-ip-addresses"
import { buildPaginationInfo, normalizePagination } from "@/hooks/_shared/pagination"
import { IPAddressService } from "@/services/ip-address.service"
import { getDateLocale } from "@/lib/date-utils"
import { escapeCSV, formatDateForCSV } from "@/lib/csv-utils"
import { downloadBlob } from "@/lib/download-utils"
import { createIPAddressColumns } from "./ip-addresses-columns"

import type { IPAddress } from "@/types/ip-address.types"

interface IPAddressesViewStateOptions {
  targetId?: number
  scanId?: number
}

export function useIPAddressesViewState({ targetId, scanId }: IPAddressesViewStateOptions) {
  const [pagination, setPagination] = React.useState({
    pageIndex: 0,
    pageSize: 10,
  })
  const [selectedIPAddresses, setSelectedIPAddresses] = React.useState<IPAddress[]>([])
  const [filterQuery, setFilterQuery] = React.useState("")
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [isDeleting, setIsDeleting] = React.useState(false)

  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const tTooltips = useTranslations("tooltips")
  const tToast = useTranslations("toast")
  const tStatus = useTranslations("common.status")
  const locale = useLocale()

  const translations = React.useMemo(
    () => ({
      columns: {
        ipAddress: tColumns("ipAddress.ipAddress"),
        hosts: tColumns("ipAddress.hosts"),
        createdAt: tColumns("common.createdAt"),
        openPorts: tColumns("ipAddress.openPorts"),
      },
      actions: {
        selectAll: tCommon("actions.selectAll"),
        selectRow: tCommon("actions.selectRow"),
      },
      tooltips: {
        allHosts: tTooltips("allHosts"),
        allOpenPorts: tTooltips("allOpenPorts"),
      },
    }),
    [tColumns, tCommon, tTooltips]
  )

  const handleFilterChange = React.useCallback((value: string) => {
    setFilterQuery(value)
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }, [])

  const targetQuery = useTargetIPAddresses(
    targetId || 0,
    {
      page: pagination.pageIndex + 1,
      pageSize: pagination.pageSize,
      filter: filterQuery || undefined,
    },
    { enabled: !!targetId }
  )

  const scanQuery = useScanIPAddresses(
    scanId || 0,
    {
      page: pagination.pageIndex + 1,
      pageSize: pagination.pageSize,
      filter: filterQuery || undefined,
    },
    { enabled: !!scanId }
  )

  const activeQuery = targetId ? targetQuery : scanQuery
  const { data, isLoading, error, refetch } = activeQuery

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
      createIPAddressColumns({
        formatDate,
        t: translations,
      }),
    [formatDate, translations]
  )

  const ipAddresses: IPAddress[] = React.useMemo(() => {
    return data?.results ?? []
  }, [data])

  const paginationInfo = data
    ? buildPaginationInfo({
        ...normalizePagination(data, pagination.pageIndex + 1, pagination.pageSize),
        minTotalPages: 1,
      })
    : undefined

  const handleSelectionChange = React.useCallback((selectedRows: IPAddress[]) => {
    setSelectedIPAddresses(selectedRows)
  }, [])

  const generateCSV = React.useCallback((items: IPAddress[]): string => {
    const bom = "\ufeff"
    const headers = ["ip", "host", "port", "created_at"]

    const rows: string[] = []
    for (const item of items) {
      for (const host of item.hosts) {
        for (const port of item.ports) {
          rows.push(
            [
              escapeCSV(item.ip),
              escapeCSV(host),
              escapeCSV(String(port)),
              escapeCSV(formatDateForCSV(item.createdAt)),
            ].join(",")
          )
        }
      }
    }

    return bom + [headers.join(","), ...rows].join("\n")
  }, [])

  const handleDownloadAll = React.useCallback(async () => {
    try {
      let blob: Blob | null = null

      if (scanId) {
        blob = await IPAddressService.exportIPAddressesByScanId(scanId)
      } else if (targetId) {
        blob = await IPAddressService.exportIPAddressesByTargetId(targetId)
      } else if (ipAddresses.length > 0) {
        const csvContent = generateCSV(ipAddresses)
        blob = new Blob([csvContent], { type: "text/csv;charset=utf-8" })
      }

      if (!blob) return

      const prefix = scanId ? `scan-${scanId}` : targetId ? `target-${targetId}` : "ip-addresses"
      downloadBlob(blob, `${prefix}-ip-addresses-${Date.now()}.csv`)
    } catch {
      toast.error(tToast("downloadFailed"))
    }
  }, [generateCSV, ipAddresses, scanId, targetId, tToast])

  const handleDownloadSelected = React.useCallback(async () => {
    if (selectedIPAddresses.length === 0) {
      return
    }

    try {
      const ips = selectedIPAddresses.map((item) => item.ip)
      let blob: Blob | null = null

      if (targetId) {
        blob = await IPAddressService.exportIPAddressesByTargetId(targetId, ips)
      } else {
        const csvContent = generateCSV(selectedIPAddresses)
        blob = new Blob([csvContent], { type: "text/csv;charset=utf-8" })
      }

      if (!blob) return

      const prefix = scanId ? `scan-${scanId}` : targetId ? `target-${targetId}` : "ip-addresses"
      downloadBlob(blob, `${prefix}-ip-addresses-selected-${Date.now()}.csv`)
    } catch {
      toast.error(tToast("downloadFailed"))
    }
  }, [generateCSV, scanId, selectedIPAddresses, targetId, tToast])

  const handleBulkDelete = React.useCallback(async () => {
    if (selectedIPAddresses.length === 0) return

    setIsDeleting(true)
    try {
      const ips = selectedIPAddresses.map((item) => item.ip)
      const result = await IPAddressService.bulkDelete(ips)
      toast.success(tToast("deleteSuccess", { count: result.deletedCount }))
      setSelectedIPAddresses([])
      setDeleteDialogOpen(false)
      refetch()
    } catch {
      toast.error(tToast("deleteFailed"))
    } finally {
      setIsDeleting(false)
    }
  }, [selectedIPAddresses, tToast, refetch])

  return {
    tCommon,
    tStatus,
    targetId,
    data,
    error,
    isLoading,
    refetch,
    columns,
    ipAddresses,
    filterQuery,
    handleFilterChange,
    pagination,
    setPagination,
    paginationInfo,
    handleSelectionChange,
    handleDownloadAll,
    handleDownloadSelected,
    handleBulkDelete,
    deleteDialogOpen,
    setDeleteDialogOpen,
    selectedIPAddresses,
    isDeleting,
  }
}

export type IPAddressesViewState = ReturnType<typeof useIPAddressesViewState>
