import React from "react"
import { useLocale, useTranslations } from "next-intl"
import { toast } from "sonner"

import { useScans } from "@/hooks/use-scans"
import { useResourceMutation } from "@/hooks/_shared/create-resource-mutation"
import { createScanHistoryColumns } from "@/components/scan/history/scan-history-columns"
import { buildScanProgressData, type ScanProgressData } from "@/components/scan/scan-progress-dialog"
import { getDateLocale } from "@/lib/date-utils"
import { buildPaginationInfo, normalizePagination } from "@/hooks/_shared/pagination"
import { getScan, deleteScan, stopScan } from "@/services/scan.service"

import type { ScanRecord } from "@/types/scan.types"
import type { PaginationInfo } from "@/types/common.types"

export function useDashboardDataTableState() {
  const t = useTranslations()
  const locale = useLocale()

  const [progressData, setProgressData] = React.useState<ScanProgressData | null>(null)
  const [progressDialogOpen, setProgressDialogOpen] = React.useState(false)

  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [scanToDelete, setScanToDelete] = React.useState<ScanRecord | null>(null)

  const [stopDialogOpen, setStopDialogOpen] = React.useState(false)
  const [scanToStop, setScanToStop] = React.useState<ScanRecord | null>(null)

  const [scanPagination, setScanPagination] = React.useState({ pageIndex: 0, pageSize: 10 })

  const scanQuery = useScans({
    page: scanPagination.pageIndex + 1,
    pageSize: scanPagination.pageSize,
  })

  const deleteMutation = useResourceMutation({
    mutationFn: deleteScan,
    invalidate: [{ queryKey: ["scans"] }],
    skipDefaultErrorHandler: true,
  })

  const stopMutation = useResourceMutation({
    mutationFn: stopScan,
    invalidate: [{ queryKey: ["scans"] }],
    skipDefaultErrorHandler: true,
  })

  const scans = scanQuery.data?.results ?? []

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

  const handleViewProgress = React.useCallback(async (scan: ScanRecord) => {
    try {
      const fullScan = await getScan(scan.id)
      const data = buildScanProgressData(fullScan)
      setProgressData(data)
      setProgressDialogOpen(true)
    } catch {
      setProgressData(buildScanProgressData(scan))
      setProgressDialogOpen(true)
    }
  }, [])

  const handleDelete = React.useCallback((scan: ScanRecord) => {
    setScanToDelete(scan)
    setDeleteDialogOpen(true)
  }, [])

  const confirmDelete = async () => {
    if (!scanToDelete) return
    setDeleteDialogOpen(false)
    try {
      await deleteMutation.mutateAsync(scanToDelete.id)
      toast.success(t("common.status.success"))
    } catch {
      toast.error(t("common.status.error"))
    } finally {
      setScanToDelete(null)
    }
  }

  const handleStop = React.useCallback((scan: ScanRecord) => {
    setScanToStop(scan)
    setStopDialogOpen(true)
  }, [])

  const confirmStop = async () => {
    if (!scanToStop) return
    setStopDialogOpen(false)
    try {
      await stopMutation.mutateAsync(scanToStop.id)
      toast.success(t("common.status.success"))
    } catch {
      toast.error(t("common.status.error"))
    } finally {
      setScanToStop(null)
    }
  }

  const scanColumns = React.useMemo(
    () =>
      createScanHistoryColumns({
        formatDate,
        handleDelete,
        handleStop,
        handleViewProgress,
        t: {
          columns: {
            target: t("columns.scanHistory.target"),
            summary: t("columns.scanHistory.summary"),
            workflowName: t("columns.scanHistory.workflowName"),
            workerName: t("columns.scanHistory.workerName"),
            createdAt: t("columns.common.createdAt"),
            status: t("columns.common.status"),
            progress: t("columns.scanHistory.progress"),
          },
          actions: {
            snapshot: t("common.actions.snapshot"),
            stop: t("scan.stopScan"),
            stopScanPending: t("scan.stopScanPending"),
            delete: t("common.actions.delete"),
            selectAll: t("common.actions.selectAll"),
            selectRow: t("common.actions.selectRow"),
          },
          tooltips: {
            targetDetails: t("tooltips.targetDetails"),
            viewProgress: t("tooltips.viewProgress"),
          },
          status: {
            cancelled: t("common.status.cancelled"),
            completed: t("common.status.completed"),
            failed: t("common.status.failed"),
            pending: t("common.status.pending"),
            running: t("common.status.running"),
          },
          summary: {
            subdomains: t("columns.scanHistory.subdomains"),
            websites: t("columns.scanHistory.websites"),
            ipAddresses: t("columns.scanHistory.ipAddresses"),
            endpoints: t("columns.scanHistory.endpoints"),
            vulnerabilities: t("columns.scanHistory.vulnerabilities"),
          },
        },
      }),
    [formatDate, handleViewProgress, handleDelete, handleStop, t]
  )

  const scanPaginationInfo: PaginationInfo = buildPaginationInfo({
    ...normalizePagination(
      scanQuery.data,
      scanPagination.pageIndex + 1,
      scanPagination.pageSize
    ),
    minTotalPages: 1,
  })

  return {
    t,
    progressData,
    progressDialogOpen,
    setProgressDialogOpen,
    deleteDialogOpen,
    setDeleteDialogOpen,
    stopDialogOpen,
    setStopDialogOpen,
    scanToDelete,
    scanToStop,
    confirmDelete,
    confirmStop,
    scans,
    scanColumns,
    scanQuery,
    scanPagination,
    setScanPagination,
    scanPaginationInfo,
  }
}

export type DashboardDataTableState = ReturnType<typeof useDashboardDataTableState>
