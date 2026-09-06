"use client"

import * as React from "react"
import { useRouter } from "next/navigation"
import { useTranslations, useLocale } from "next-intl"
import { ScheduledScanDataTable } from "@/components/scan/scheduled/scheduled-scan-data-table"
import { createScheduledScanColumns } from "@/components/scan/scheduled/scheduled-scan-columns"
import { useScheduledScans } from "@/hooks/use-scheduled-scans"
import { useSearchState } from "@/hooks/_shared/use-search-state"
import { buildPaginationInfo } from "@/hooks/_shared/pagination"
import { getDataTableSkeletonRowCount } from "@/components/shared/loading/data-table-skeleton"
import { getLoadingOwnerAttributes } from "@/components/shared/loading/loading-owner"
import { pushWithRouteProgress } from "@/components/route-progress"
import { getDateLocale } from "@/lib/date-utils"
import type { ScheduledScan } from "@/types/scheduled-scan.types"
import type { ColumnDef } from "@tanstack/react-table"

function OverviewScheduledScansTableLoadingState({
  columns,
  pagination,
  searchPlaceholder,
}: {
  columns: ColumnDef<ScheduledScan>[]
  pagination: { page: number; pageSize: number }
  searchPlaceholder: string
}) {
  return (
    <div
      {...getLoadingOwnerAttributes({ owner: "overview-scheduled-scans", layer: "section", intent: "data" })}
      data-slot="overview-scheduled-scans-table-loading-state"
      className="w-full"
    >
      <ScheduledScanDataTable
        data={[]}
        columns={columns}
        searchPlaceholder={searchPlaceholder}
        page={pagination.page}
        pageSize={pagination.pageSize}
        total={0}
        totalPages={1}
        onPageChange={() => {}}
        onPageSizeChange={() => {}}
        showQuickFilters={false}
        loading
        loadingRowCount={getDataTableSkeletonRowCount(pagination.pageSize)}
      />
    </div>
  )
}

export function OverviewScheduledScans() {
  const [pagination, setPagination] = React.useState({ page: 1, pageSize: 10 })
  const [searchQuery, setSearchQuery] = React.useState("")
  const router = useRouter()
  const locale = useLocale()

  // Internationalization
  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const tScan = useTranslations("scan")

  // Build translation object
  const translations = React.useMemo(() => ({
    columns: {
      taskName: tColumns("scheduledScan.taskName"),
      cronExpression: tColumns("scheduledScan.cronExpression"),
      scope: tColumns("scheduledScan.scope"),
      status: tColumns("common.status"),
      nextRun: tColumns("scheduledScan.nextRun"),
      handoffResults: tColumns("scheduledScan.handoffResults"),
      trigger: tColumns("scheduledScan.trigger"),
      success: tColumns("scheduledScan.success"),
      failure: tColumns("scheduledScan.failure"),
      lastRun: tColumns("scheduledScan.lastRun"),
    },
    actions: {
      editTask: tScan("editTask"),
      delete: tCommon("actions.delete"),
      openMenu: tCommon("actions.openMenu"),
      selectAll: tCommon("actions.selectAll"),
      selectRow: tCommon("actions.selectRow"),
    },
    status: {
      enabled: tCommon("status.enabled"),
      disabled: tCommon("status.disabled"),
    },
    cron: {
      everyMinute: tScan("cron.everyMinute"),
      everyNMinutes: tScan.raw("cron.everyNMinutes") as string,
      everyHour: tScan.raw("cron.everyHour") as string,
      everyNHours: tScan.raw("cron.everyNHours") as string,
      everyDay: tScan.raw("cron.everyDay") as string,
      everyWeek: tScan.raw("cron.everyWeek") as string,
      everyMonth: tScan.raw("cron.everyMonth") as string,
      weekdays: tScan.raw("cron.weekdays") as string[],
    },
  }), [tColumns, tCommon, tScan])

  const { data, isLoading, isFetching } = useScheduledScans({
    page: pagination.page,
    pageSize: pagination.pageSize,
    search: searchQuery || undefined,
  })
  const { isSearching, commitSearch } = useSearchState({
    isFetching,
    searchValue: searchQuery,
    setSearchValue: setSearchQuery,
    onResetPage: () => setPagination((prev) => ({ ...prev, page: 1 })),
  })

  const formatDate = React.useCallback(
    (dateString: string) => new Date(dateString).toLocaleString(getDateLocale(locale), { hour12: false }),
    [locale]
  )
  const handleEdit = React.useCallback(() => pushWithRouteProgress(router, `/scan/scheduled/`), [router])
  const handleDelete = React.useCallback(() => {}, [])
  const handleToggleStatus = React.useCallback(() => {}, [])

  const columns = React.useMemo(
    () =>
      createScheduledScanColumns({
        formatDate,
        handleEdit,
        handleDelete,
        handleToggleStatus,
        t: translations,
      }),
    [formatDate, handleEdit, handleDelete, handleToggleStatus, translations]
  )
  const list = data?.scheduledScans ?? []

  if (isLoading && !data) {
    return (
      <OverviewScheduledScansTableLoadingState
        columns={columns}
        pagination={pagination}
        searchPlaceholder={tScan("scheduled.searchPlaceholder")}
      />
    )
  }

  const paginationInfo = buildPaginationInfo({
    total: data?.totalSize ?? 0,
    page: pagination.page,
    pageSize: pagination.pageSize,
    minTotalPages: 1,
  })

  return (
    <ScheduledScanDataTable
      data={list}
      columns={columns}
      searchPlaceholder={tScan("scheduled.searchPlaceholder")}
      searchValue={searchQuery}
      onSearch={commitSearch}
      isSearching={isSearching}
      page={pagination.page}
      pageSize={pagination.pageSize}
      total={paginationInfo.total}
      totalPages={paginationInfo.totalPages}
      onPageChange={(page) => setPagination((prev) => ({ ...prev, page }))}
      onPageSizeChange={(pageSize) => setPagination({ page: 1, pageSize })}
      showQuickFilters={false}
    />
  )
}
