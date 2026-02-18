"use client"

import * as React from "react"
import { useRouter } from "next/navigation"
import { useTranslations, useLocale } from "next-intl"
import { ScheduledScanDataTable } from "@/components/scan/scheduled/scheduled-scan-data-table"
import { createScheduledScanColumns } from "@/components/scan/scheduled/scheduled-scan-columns"
import { useScheduledScans } from "@/hooks/use-scheduled-scans"
import { useSearchState } from "@/hooks/_shared/use-search-state"
import { buildPaginationInfo } from "@/hooks/_shared/pagination"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"
import { getDateLocale } from "@/lib/date-utils"

export function DashboardScheduledScans() {
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
      scanEngine: tColumns("scheduledScan.scanEngine"),
      cronExpression: tColumns("scheduledScan.cronExpression"),
      scope: tColumns("scheduledScan.scope"),
      status: tColumns("common.status"),
      nextRun: tColumns("scheduledScan.nextRun"),
      runCount: tColumns("scheduledScan.runCount"),
      lastRun: tColumns("scheduledScan.lastRun"),
    },
    actions: {
      editTask: tScan("editTask"),
      delete: tCommon("actions.delete"),
      openMenu: tCommon("actions.openMenu"),
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
  const { isSearching, handleSearchChange } = useSearchState({
    isFetching,
    setSearchValue: setSearchQuery,
    onResetPage: () => setPagination((prev) => ({ ...prev, page: 1 })),
  })

  const formatDate = React.useCallback(
    (dateString: string) => new Date(dateString).toLocaleString(getDateLocale(locale), { hour12: false }),
    [locale]
  )
  const handleEdit = React.useCallback(() => router.push(`/scan/scheduled/`), [router])
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

  const list = data?.results ?? []
  const paginationInfo = buildPaginationInfo({
    total: data?.total ?? 0,
    page: pagination.page,
    pageSize: pagination.pageSize,
    totalPages: data?.totalPages,
    minTotalPages: 1,
  })

  return (
    <ScheduledScanDataTable
      data={list}
      columns={columns}
      searchPlaceholder={tScan("scheduled.searchPlaceholder")}
      searchValue={searchQuery}
      onSearch={handleSearchChange}
      isSearching={isSearching}
      page={pagination.page}
      pageSize={pagination.pageSize}
      total={paginationInfo.total}
      totalPages={paginationInfo.totalPages}
      onPageChange={(page) => setPagination((prev) => ({ ...prev, page }))}
      onPageSizeChange={(pageSize) => setPagination({ page: 1, pageSize })}
    />
  )
}
