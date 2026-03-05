"use client"

import * as React from "react"
import { useTranslations, useLocale } from "next-intl"
import { ScanHistoryDataTable } from "@/components/scan/history/scan-history-data-table"
import { createScanHistoryColumns } from "@/components/scan/history/scan-history-columns"
import { useScans } from "@/hooks/use-scans"
import { buildPaginationInfo, normalizePagination } from "@/hooks/_shared/pagination"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"
import { getDateLocale } from "@/lib/date-utils"
import type { ScanRecord } from "@/types/scan.types"
import type { ColumnDef } from "@tanstack/react-table"

export function DashboardScanHistory() {
  const [pagination, setPagination] = React.useState({ pageIndex: 0, pageSize: 5 })
  const locale = useLocale()

  // i18n
  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const tTooltips = useTranslations("tooltips")
  const tScan = useTranslations("scan")

  // Build translation map
  const translations = React.useMemo(() => ({
    columns: {
      target: tColumns("scanHistory.target"),
      summary: tColumns("scanHistory.summary"),
      workflowName: tColumns("scanHistory.workflowName"),
      workerName: tColumns("scanHistory.workerName"),
      createdAt: tColumns("common.createdAt"),
      status: tColumns("common.status"),
      progress: tColumns("scanHistory.progress"),
    },
    actions: {
      snapshot: tCommon("actions.snapshot"),
      stop: tCommon("actions.stop"),
      stopScanPending: tScan("stopScanPending"),
      delete: tCommon("actions.delete"),
      selectAll: tCommon("actions.selectAll"),
      selectRow: tCommon("actions.selectRow"),
    },
    tooltips: {
      targetDetails: tTooltips("targetDetails"),
      viewProgress: tTooltips("viewProgress"),
    },
    status: {
      cancelled: tCommon("status.cancelled"),
      completed: tCommon("status.completed"),
      failed: tCommon("status.failed"),
      pending: tCommon("status.pending"),
      running: tCommon("status.running"),
    },
    summary: {
      subdomains: tColumns("scanHistory.subdomains"),
      websites: tColumns("scanHistory.websites"),
      ipAddresses: tColumns("scanHistory.ipAddresses"),
      endpoints: tColumns("scanHistory.endpoints"),
      vulnerabilities: tColumns("scanHistory.vulnerabilities"),
    },
  }), [tColumns, tCommon, tTooltips, tScan])

  const { data, isLoading } = useScans({
    page: pagination.pageIndex + 1,
    pageSize: pagination.pageSize,
    status: 'running',
  })

  const formatDate = React.useCallback((dateString: string) => new Date(dateString).toLocaleString(getDateLocale(locale), { hour12: false }), [locale])
  const handleDelete = React.useCallback(() => {}, [])
  const handleStop = React.useCallback(() => {
    // Stop action is not wired for the dashboard list yet.
    // Hook the stop-scan API here when the feature is enabled.
  }, [])

  const columns = React.useMemo(
    () => createScanHistoryColumns({ formatDate, handleDelete, handleStop, t: translations }) as ColumnDef<ScanRecord>[],
    [formatDate, handleDelete, handleStop, translations]
  )

  if (isLoading && !data) {
    return (
      <DataTableSkeleton
        withPadding={false}
        toolbarButtonCount={2}
        rows={4}
        columns={3}
      />
    )
  }

  const paginationInfo = data
    ? buildPaginationInfo({
      ...normalizePagination(data, pagination.pageIndex + 1, pagination.pageSize),
      minTotalPages: 1,
    })
    : undefined

  return (
    <ScanHistoryDataTable
      data={data?.results ?? []}
      columns={columns}
      hideToolbar
      hidePagination
      pagination={pagination}
      setPagination={setPagination}
      paginationInfo={paginationInfo}
      onPaginationChange={setPagination}
    />
  )
}
