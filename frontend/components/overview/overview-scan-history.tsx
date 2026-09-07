"use client"

import * as React from "react"
import { useTranslations, useLocale } from "next-intl"
import { ScanHistoryDataTable } from "@/components/scan/history/scan-history-data-table"
import { createScanHistoryColumns } from "@/components/scan/history/scan-history-columns"
import { useScans } from "@/hooks/use-scans"
import { useScanExecutedEngineDisplay } from "@/hooks/use-scan-executed-engine-display"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { normalizeError } from "@/lib/errors/normalize-error"
import { buildPaginationInfo, normalizePagination } from "@/hooks/_shared/pagination"
import { getDataTableSkeletonRowCount } from "@/components/shared/loading/data-table-skeleton"
import { getLoadingOwnerAttributes } from "@/components/shared/loading/loading-owner"
import { getDateLocale } from "@/lib/date-utils"
import type { ScanRecord } from "@/types/scan.types"
import type { ColumnDef } from "@tanstack/react-table"

function OverviewScanHistoryTableLoadingState({
  columns,
  pagination,
}: {
  columns: ColumnDef<ScanRecord>[]
  pagination: { pageIndex: number; pageSize: number }
}) {
  return (
    <div
      {...getLoadingOwnerAttributes({ owner: "overview-scan-history", layer: "section", intent: "data" })}
      data-slot="overview-scan-history-table-loading-state"
      className="w-full"
    >
      <ScanHistoryDataTable
        data={[]}
        columns={columns}
        hideToolbar
        hidePagination
        pagination={pagination}
        loading
        loadingRowCount={getDataTableSkeletonRowCount(pagination.pageSize)}
      />
    </div>
  )
}

export function OverviewScanHistory() {
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
      executedEngines: tColumns("scanHistory.executedEngines"),
      triggerType: tColumns("scanHistory.triggerType"),
      createdAt: tColumns("common.createdAt"),
      status: tColumns("common.status"),
      progress: tColumns("scanHistory.progress"),
    },
    actions: {
      scanDetail: tScan("history.actions.scanDetail"),
      runtimeDetail: tScan("history.runtimeDrawer.open"),
      openMenu: tCommon("actions.openMenu"),
      stop: tCommon("actions.stop"),
      stopScanPending: tScan("stopScanPending"),
      delete: tCommon("actions.delete"),
      selectAll: tCommon("actions.selectAll"),
      selectRow: tCommon("actions.selectRow"),
    },
    tooltips: {
      viewProgress: tTooltips("viewProgress"),
    },
    status: {
      cancelled: tCommon("status.cancelled"),
      succeeded: tCommon("status.succeeded"),
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
    triggerTypes: {
      manual: tScan("history.triggerType.manual"),
      scheduled: tScan("history.triggerType.scheduled"),
      ai: tScan("history.triggerType.ai"),
    },
  }), [tColumns, tCommon, tTooltips, tScan])

  const { data, isLoading, error, refetch } = useScans({
    page: pagination.pageIndex + 1,
    pageSize: pagination.pageSize,
    status: 'running',
  })
  const scans = data?.results ?? []
  const executedEngineDisplay = useScanExecutedEngineDisplay(scans)

  const formatDate = React.useCallback((dateString: string) => new Date(dateString).toLocaleString(getDateLocale(locale), { hour12: false }), [locale])
  const handleDelete = React.useCallback(() => {}, [])
  const handleStop = React.useCallback(() => {
    // Stop action is not wired for the overview list yet.
    // Hook the stop-scan API here when the feature is enabled.
  }, [])

  const columns = React.useMemo(
    () => createScanHistoryColumns({
      formatDate,
      handleDelete,
      handleStop,
      t: translations,
      executedEngineNamesByScanId: executedEngineDisplay.engineNamesByScanId,
      executedEngineDescriptionsByScanId: executedEngineDisplay.engineDescriptionsByScanId,
    }) as ColumnDef<ScanRecord>[],
    [formatDate, handleDelete, handleStop, translations, executedEngineDisplay.engineDescriptionsByScanId, executedEngineDisplay.engineNamesByScanId]
  )

  if (isLoading || executedEngineDisplay.isLoading) {
    return <OverviewScanHistoryTableLoadingState columns={columns} pagination={pagination} />
  }

  const queryError = error ?? executedEngineDisplay.error

  if (queryError) {
    return (
      <AppErrorState
        error={normalizeError(queryError, { notFoundKind: "unexpected-error" })}
        title={tScan("progress.engineCatalogUnavailable")}
        onRetry={() => Promise.all([refetch(), executedEngineDisplay.refetch()])}
        variant="section"
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
      data={scans}
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
