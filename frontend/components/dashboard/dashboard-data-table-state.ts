import React from "react"
import { useLocale, useTranslations } from "next-intl"
import { toast } from "sonner"

import { useAllVulnerabilities } from "@/hooks/use-vulnerabilities"
import { useScans } from "@/hooks/use-scans"
import { useResourceMutation } from "@/hooks/_shared/create-resource-mutation"
import { createVulnerabilityColumns } from "@/components/vulnerabilities/vulnerabilities-columns"
import { createScanHistoryColumns } from "@/components/scan/history/scan-history-columns"
import { buildScanProgressData, type ScanProgressData } from "@/components/scan/scan-progress-dialog"
import { getDateLocale } from "@/lib/date-utils"
import { buildPaginationInfo, normalizePagination } from "@/hooks/_shared/pagination"
import { getScan, deleteScan, stopScan } from "@/services/scan.service"

import type { Vulnerability } from "@/types/vulnerability.types"
import type { ScanRecord } from "@/types/scan.types"
import type { PaginationInfo } from "@/types/common.types"

export function useDashboardDataTableState() {
  const t = useTranslations()
  const locale = useLocale()

  const [activeTab, setActiveTab] = React.useState("scans")

  const [selectedVuln, setSelectedVuln] = React.useState<Vulnerability | null>(null)
  const [vulnDialogOpen, setVulnDialogOpen] = React.useState(false)

  const [progressData, setProgressData] = React.useState<ScanProgressData | null>(null)
  const [progressDialogOpen, setProgressDialogOpen] = React.useState(false)

  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [scanToDelete, setScanToDelete] = React.useState<ScanRecord | null>(null)

  const [stopDialogOpen, setStopDialogOpen] = React.useState(false)
  const [scanToStop, setScanToStop] = React.useState<ScanRecord | null>(null)

  const [vulnPagination, setVulnPagination] = React.useState({ pageIndex: 0, pageSize: 10 })
  const [scanPagination, setScanPagination] = React.useState({ pageIndex: 0, pageSize: 10 })

  const vulnQuery = useAllVulnerabilities({
    page: vulnPagination.pageIndex + 1,
    pageSize: vulnPagination.pageSize,
  })

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

  const vulnerabilities = vulnQuery.data?.vulnerabilities ?? []
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

  const handleVulnRowClick = React.useCallback((vuln: Vulnerability) => {
    setSelectedVuln(vuln)
    setVulnDialogOpen(true)
  }, [])

  const vulnColumns = React.useMemo(
    () =>
      createVulnerabilityColumns({
        formatDate,
        handleViewDetail: handleVulnRowClick,
        t: {
          columns: {
            severity: t("columns.vulnerability.severity"),
            source: t("columns.vulnerability.source"),
            vulnType: t("columns.vulnerability.vulnType"),
            url: t("columns.common.url"),
            createdAt: t("columns.common.createdAt"),
          },
          actions: {
            details: t("common.actions.details"),
            selectAll: t("common.actions.selectAll"),
            selectRow: t("common.actions.selectRow"),
          },
          tooltips: {
            vulnDetails: t("tooltips.vulnDetails"),
            reviewed: t("tooltips.reviewed"),
            pending: t("tooltips.pending"),
          },
          severity: {
            critical: t("severity.critical"),
            high: t("severity.high"),
            medium: t("severity.medium"),
            low: t("severity.low"),
            info: t("severity.info"),
          },
        },
      }),
    [formatDate, handleVulnRowClick, t]
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
            engineName: t("columns.scanHistory.engineName"),
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

  const vulnPaginationInfo: PaginationInfo = buildPaginationInfo({
    ...normalizePagination(
      vulnQuery.data?.pagination,
      vulnPagination.pageIndex + 1,
      vulnPagination.pageSize
    ),
    minTotalPages: 1,
  })

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
    activeTab,
    setActiveTab,
    selectedVuln,
    vulnDialogOpen,
    setVulnDialogOpen,
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
    vulnerabilities,
    scans,
    vulnColumns,
    scanColumns,
    vulnQuery,
    scanQuery,
    vulnPagination,
    setVulnPagination,
    scanPagination,
    setScanPagination,
    vulnPaginationInfo,
    scanPaginationInfo,
  }
}

export type DashboardDataTableState = ReturnType<typeof useDashboardDataTableState>
