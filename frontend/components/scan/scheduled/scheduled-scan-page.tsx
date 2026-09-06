"use client"

import React from "react"
import dynamic from "next/dynamic"
import { semanticIcons } from "@/components/icons"
import type { SelectedRowActionBarAction } from "@/components/shared/data-table"
import { ScheduledScanDataTable } from "@/components/scan/scheduled/scheduled-scan-data-table"
import { useLocale, useTranslations } from "next-intl"
import { createScheduledScanColumns } from "@/components/scan/scheduled/scheduled-scan-columns"
import {
	ScheduledScanInsightGrid,
	ScheduledScanOverviewFailure,
	ScheduledScanOverviewLoadingState,
  ScheduledScanOverviewStaleNotice,
  ScheduledScanPageHeader,
  ScheduledScanPageLoadingState,
  ScheduledScanTableSectionShell,
  ScheduledScanTimeline,
} from "@/components/scan/scheduled/scheduled-scan-page-sections"
import {
  AlertDialog,
  AlertDialogClose, AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { 
  useScheduledScans, 
  useBatchDeleteScheduledScans,
	useBatchUpdateScheduledScanStatus,
  useDeleteScheduledScan, 
  useToggleScheduledScan,
  useScheduledScanOverviewSummary,
} from "@/hooks/use-scheduled-scans"
import type { ScheduledScan } from "@/types/scheduled-scan.types"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { getDataTableSkeletonRowCount } from "@/components/shared/loading/data-table-skeleton"
import { useInteractionOpenLoader } from "@/components/shared/loading/use-interaction-open-loader"
import {
  deferredInteractionUnmountDelayMs,
  useDeferredInteractionMount,
} from "@/hooks/use-deferred-interaction-mount"
import { useSearchState } from "@/hooks/_shared/use-search-state"
import type { ScheduledScanQuickFilter } from "@/components/scan/scheduled/scheduled-scan-data-table"
import { SCHEDULED_SCAN_PAGE_SIZE } from "@/components/scan/scheduled/scheduled-scan-page-layout"
import {
  getCurrentCursorNextPageToken,
  getCursorPageTransition,
  getCursorPaginationNavigation,
} from "@/components/shared/data-table/business-list-query"

export interface ScheduledScanPageProps {
  onReady?: () => void
  deferInitialSkeleton?: boolean
}

const loadCreateScheduledScanSheet = () =>
  import("@/components/scan/scheduled/create-scheduled-scan-dialog").then((mod) => mod.CreateScheduledScanSheet)

const CreateScheduledScanSheet = dynamic(
  () => loadCreateScheduledScanSheet(),
  { ssr: false }
)

const EditScheduledScanDialog = dynamic(
  () =>
    import("@/components/scan/scheduled/edit-scheduled-scan-dialog").then(
      (mod) => mod.EditScheduledScanDialog
    ),
  { ssr: false, loading: () => null }
)

const EMPTY_SCHEDULED_SCANS: ScheduledScan[] = []
const EnableScheduledScansIcon = semanticIcons.status.enabled
const DisableScheduledScansIcon = semanticIcons.status.disabled

function filterScheduledScansByQuickFilter(
  scans: ScheduledScan[],
  filter: ScheduledScanQuickFilter
) {
  if (filter === "all") return scans

  return scans.filter((scan) => {
    if (filter === "enabled") return scan.isEnabled
    if (filter === "paused") return !scan.isEnabled
    return true
  })
}

/**
 * Scheduled scan page
 * Manage scheduled scan task configuration
 */
export default function ScheduledScanPage({
  onReady,
  deferInitialSkeleton = false,
}: ScheduledScanPageProps) {
  const [isRouteBoundaryEntry] = React.useState(deferInitialSkeleton)
  const {
    openAfterLoad,
  } = useInteractionOpenLoader<"create-dialog">()
  const [createDialogOpen, setCreateDialogOpen] = React.useState(false)
  const [editDialogOpen, setEditDialogOpen] = React.useState(false)
  const [editingScheduledScan, setEditingScheduledScan] = React.useState<ScheduledScan | null>(null)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [deletingScheduledScan, setDeletingScheduledScan] = React.useState<ScheduledScan | null>(null)
  const [selectedScheduledScans, setSelectedScheduledScans] = React.useState<ScheduledScan[]>([])
  const [batchDisableDialogOpen, setBatchDisableDialogOpen] = React.useState(false)
  const sidebarMountOptions = { unmountDelayMs: deferredInteractionUnmountDelayMs }
  const shouldMountCreateDialog = useDeferredInteractionMount(createDialogOpen, sidebarMountOptions)
  const shouldMountEditDialog = useDeferredInteractionMount(editDialogOpen, sidebarMountOptions)
  
  // Internationalization
  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const tScan = useTranslations("scan")
  const tConfirm = useTranslations("common.confirm")
  const locale = useLocale()

  // Build translation object
  const translations = React.useMemo(() => ({
    columns: {
      taskName: tColumns("scheduledScan.taskName"),
      cronExpression: tColumns("scheduledScan.cronExpression"),
      scope: tScan("scheduled.workbench.targetColumn"),
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
  
  // Pagination state
  const [page, setPage] = React.useState(1)
  const [pageSize, setPageSize] = React.useState(SCHEDULED_SCAN_PAGE_SIZE)
  // Opaque tokens only authorize adjacent pages returned by this query; they cannot be derived from totalSize.
  const [pageTokens, setPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })

  // Search state
  const [searchQuery, setSearchQuery] = React.useState("")
  const [quickFilter, setQuickFilter] = React.useState<ScheduledScanQuickFilter>("all")
  
  // Use actual API
  const pageToken = page === 1 ? undefined : pageTokens[page]
  const { data, isLoading, isFetching, isPlaceholderData, refetch } = useScheduledScans({
    pageSize,
    pageToken,
    search: searchQuery || undefined,
  })
  const overview = useScheduledScanOverviewSummary()

  const { isSearching, commitSearch } = useSearchState({
    isFetching,
    searchValue: searchQuery,
    setSearchValue: setSearchQuery,
    onResetPage: () => {
      setPageTokens({ 1: undefined })
      setPage(1)
    },
  })
  const { mutate: deleteScheduledScan } = useDeleteScheduledScan()
  const { mutateAsync: batchDeleteScheduledScans } = useBatchDeleteScheduledScans()
  const {
    mutateAsync: batchUpdateScheduledScanStatus,
    isPending: isBatchStatusUpdatePending,
  } = useBatchUpdateScheduledScanStatus()
  const { mutate: toggleScheduledScan } = useToggleScheduledScan()

  const scheduledScans = data?.scheduledScans ?? EMPTY_SCHEDULED_SCANS
  const nextPageToken = getCurrentCursorNextPageToken(data?.nextPageToken, isPlaceholderData)
  const quickFilterCounts = React.useMemo(
    () => {
      if (overview.data) {
        return {
          all: overview.data.enabledScheduledScanCount + overview.data.pausedScheduledScanCount,
          enabled: overview.data.enabledScheduledScanCount,
          paused: overview.data.pausedScheduledScanCount,
        }
      }
      return scheduledScans.reduce<Record<ScheduledScanQuickFilter, number>>(
        (counts, scan) => {
          counts.all += 1
          if (scan.isEnabled) {
            counts.enabled += 1
          } else {
            counts.paused += 1
          }
          return counts
        },
        { all: 0, enabled: 0, paused: 0 }
      )
    }, [overview.data, scheduledScans]
  )
  const displayedScheduledScans = React.useMemo(
    () => filterScheduledScansByQuickFilter(scheduledScans, quickFilter),
    [scheduledScans, quickFilter]
  )
  const displayedTotal = quickFilter === "all" ? (data?.totalSize ?? 0) : quickFilterCounts[quickFilter]
  const paginationNavigation = React.useMemo(
    () => getCursorPaginationNavigation({
      currentPage: page,
      pageTokens,
      nextPageToken,
    }),
    [nextPageToken, page, pageTokens]
  )

  React.useEffect(() => {
    if (!nextPageToken) return
    setPageTokens((tokens) => ({ ...tokens, [page + 1]: nextPageToken }))
  }, [nextPageToken, page])

  // Format date
  const formatDate = React.useCallback((dateString: string) => {
    const date = new Date(dateString)
    return date.toLocaleString(locale, {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    })
  }, [locale])

  const formatTimelineTime = React.useCallback((dateString: string) => {
    const date = new Date(dateString)
    return date.toLocaleTimeString(locale, {
      hour: "2-digit",
      minute: "2-digit",
      hour12: false,
    })
  }, [locale])

  const formatTimelineDate = React.useCallback((dateString: string) => {
    const date = new Date(dateString)
    return date.toLocaleDateString(locale, {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
    })
  }, [locale])

  // Edit task
  const handleEdit = React.useCallback((scan: ScheduledScan) => {
    setEditingScheduledScan(scan)
    setEditDialogOpen(true)
  }, [])

  // Delete task (open confirmation dialog)
  const handleDelete = React.useCallback((scan: ScheduledScan) => {
    setDeletingScheduledScan(scan)
    setDeleteDialogOpen(true)
  }, [])

  // Confirm delete task
  const confirmDelete = React.useCallback(() => {
    if (deletingScheduledScan) {
      deleteScheduledScan(deletingScheduledScan.id)
      setDeleteDialogOpen(false)
      setDeletingScheduledScan(null)
    }
  }, [deletingScheduledScan, deleteScheduledScan])

  const handleBulkDelete = React.useCallback(async () => {
    const ids = selectedScheduledScans.map((scan) => scan.id)
    if (ids.length === 0) return

    await batchDeleteScheduledScans(ids)
    setSelectedScheduledScans([])
    refetch()
	}, [batchDeleteScheduledScans, refetch, selectedScheduledScans])

  const handleBatchStatusUpdate = React.useCallback(async (isEnabled: boolean) => {
    const ids = selectedScheduledScans.map((scan) => scan.id)
    if (ids.length === 0) return

    try {
      await batchUpdateScheduledScanStatus({ ids, isEnabled })
      setSelectedScheduledScans([])
    } catch {
      // The mutation owns error feedback; retaining selection preserves retry context.
    }
  }, [batchUpdateScheduledScanStatus, selectedScheduledScans])

  const selectedRowActions = React.useMemo<SelectedRowActionBarAction[]>(() => [
    {
      key: "enable-scheduled-scans",
      label: tScan("scheduled.batchStatus.enable"),
      icon: EnableScheduledScansIcon,
      tone: "success",
      group: "status",
      disabled: isBatchStatusUpdatePending,
      onClick: () => {
        void handleBatchStatusUpdate(true)
      },
    },
    {
      key: "disable-scheduled-scans",
      label: tScan("scheduled.batchStatus.disable"),
      icon: DisableScheduledScansIcon,
      tone: "muted",
      group: "status",
      disabled: isBatchStatusUpdatePending,
      onClick: () => setBatchDisableDialogOpen(true),
    },
  ], [handleBatchStatusUpdate, isBatchStatusUpdatePending, tScan])

  // Toggle task enabled status
  const handleToggleStatus = React.useCallback((scan: ScheduledScan, enabled: boolean) => {
    toggleScheduledScan({ id: scan.id, isEnabled: enabled })
  }, [toggleScheduledScan])

  // Page change handler
  const handlePageChange = React.useCallback((newPage: number) => {
    if (newPage === 1) {
      setPageTokens({ 1: undefined })
      setPage(1)
      return
    }
    const transition = getCursorPageTransition({
      currentPage: page,
      pageTokens,
      nextPageToken,
      requestedPage: newPage,
    })
    if (!transition.reachable) return
    setPageTokens((tokens) => ({ ...tokens, [newPage]: transition.pageToken }))
    setPage(newPage)
  }, [nextPageToken, page, pageTokens])

  // Page size change handler
  const handlePageSizeChange = React.useCallback((newPageSize: number) => {
    setPageTokens({ 1: undefined })
    setPageSize(newPageSize)
    setPage(1)
  }, [])

  const handleQuickFilterChange = React.useCallback((filter: ScheduledScanQuickFilter) => {
    setPageTokens({ 1: undefined })
    setQuickFilter(filter)
    setPage(1)
  }, [])

  // Add new task
  const handleAddNew = React.useCallback(() => {
    void openAfterLoad("create-dialog", loadCreateScheduledScanSheet, () => {
      setCreateDialogOpen(true)
    })
  }, [openAfterLoad])

  // Create column definition
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

  const hasPrimaryDataFrame = !isLoading
  const stableSurfaceRowCount = getDataTableSkeletonRowCount(pageSize || SCHEDULED_SCAN_PAGE_SIZE)
	const lastSuccessfulOverviewTime = overview.lastSuccessfulAt === null
		? null
		: new Date(overview.lastSuccessfulAt).toLocaleString(locale, {
			year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit",
		})

  React.useEffect(() => {
    if (!hasPrimaryDataFrame) return
    onReady?.()
  }, [hasPrimaryDataFrame, onReady])

  if (isLoading && deferInitialSkeleton) return null

  const pageContent = (
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <ScheduledScanPageHeader
        title={tScan("scheduled.title")}
        description={tScan("scheduled.description")}
      />

	{overview.isPending && !overview.data ? <ScheduledScanOverviewLoadingState /> : null}
	{overview.isInitialError ? (
		<ScheduledScanOverviewFailure
			title={tScan("scheduled.workbench.overview.loadFailed")}
			description={tScan("scheduled.workbench.overview.loadFailedDescription")}
			retryLabel={tScan("scheduled.workbench.overview.retry")}
			onRetry={() => { void overview.refetch() }}
		/>
	) : null}
	{overview.data ? (
		<>
			<ScheduledScanInsightGrid>
				{overview.isRefetchStale && lastSuccessfulOverviewTime ? (
					<ScheduledScanOverviewStaleNotice
						description={tScan("scheduled.workbench.overview.stale.description", { time: lastSuccessfulOverviewTime })}
						retryLabel={tScan("scheduled.workbench.overview.stale.retry")}
						onRetry={() => { void overview.refetch() }}
					/>
				) : null}
				<ScheduledScanTimeline
					items={overview.data.upcomingScheduledScans}
					labels={{
						title: tScan("scheduled.workbench.timeline.title"),
						next24HoursCount: tScan("scheduled.workbench.timeline.next24HoursCount", {
							count: overview.data.next24HoursScheduledScanCount,
						}),
						soon: tScan("scheduled.workbench.timeline.soon"),
						empty: tScan("scheduled.workbench.timeline.empty"),
					}}
					formatTime={formatTimelineTime}
					formatDate={formatTimelineDate}
				/>
			</ScheduledScanInsightGrid>
		</>
	) : null}

      {/* Data table */}
      <ScheduledScanTableSectionShell>
        <ScheduledScanDataTable
          data={displayedScheduledScans}
          columns={columns}
          onAddNew={handleAddNew}
          onBulkDelete={handleBulkDelete}
          selectedRowActions={selectedRowActions}
          onRowClick={handleEdit}
          onSelectionChange={setSelectedScheduledScans}
          selectedRows={selectedScheduledScans}
          searchPlaceholder={tScan("scheduled.searchPlaceholder")}
          searchValue={searchQuery}
          onSearch={commitSearch}
          isSearching={isSearching}
          addButtonText={tScan("scheduled.createTitle")}
          page={page}
          pageSize={pageSize}
          total={displayedTotal}
          cursorPaginationSummary={{ total: displayedTotal }}
          paginationNavigation={paginationNavigation}
          onPageChange={handlePageChange}
          onPageSizeChange={handlePageSizeChange}
          quickFilter={quickFilter}
          onQuickFilterChange={handleQuickFilterChange}
          quickFilterCounts={quickFilterCounts}
          quickFilterLabels={{
            all: tCommon("actions.all"),
            enabled: tScan("scheduled.workbench.filters.enabled"),
            paused: tScan("scheduled.workbench.filters.paused"),
          }}
          stableSurfaceRowCount={stableSurfaceRowCount}
        />
      </ScheduledScanTableSectionShell>

      {/* Create scheduled scan sheet */}
      {shouldMountCreateDialog ? (
        <CreateScheduledScanSheet
          open={createDialogOpen}
          onOpenChange={setCreateDialogOpen}
          onSuccess={() => {
            refetch()
          }}
        />
      ) : null}

      {/* Edit scheduled scan drawer */}
      {shouldMountEditDialog ? (
        <EditScheduledScanDialog
          open={editDialogOpen}
          onOpenChange={setEditDialogOpen}
          scheduledScan={editingScheduledScan}
          onSuccess={() => {
            refetch()
          }}
        />
      ) : null}

      {/* Delete confirmation dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{tConfirm("deleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {tConfirm("deleteScheduledScanMessage", { name: deletingScheduledScan?.name ?? "" })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose variant="outline">{tCommon("actions.cancel")}</AlertDialogClose>
            <AlertDialogClose onClick={confirmDelete} className="bg-destructive hover:bg-destructive/90 text-destructive-foreground">
              {tCommon("actions.delete")}
            </AlertDialogClose>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={batchDisableDialogOpen} onOpenChange={setBatchDisableDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{tScan("scheduled.batchStatus.disableTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {tScan("scheduled.batchStatus.disableDescription", {
                count: selectedScheduledScans.length,
              })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose variant="outline">{tCommon("actions.cancel")}</AlertDialogClose>
            <AlertDialogClose
              variant="destructive"
              disabled={isBatchStatusUpdatePending}
              onClick={() => {
                void handleBatchStatusUpdate(false)
              }}
            >
              {tScan("scheduled.batchStatus.disable")}
            </AlertDialogClose>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )

  // The route boundary owns the cold-entry skeleton. Keeping its resolved DOM
  // direct preserves the same geometry slots after readiness is reported.
  if (isRouteBoundaryEntry) {
    return pageContent
  }

  return (
    <ContentHandoff
      owner="scheduled-scan-page-content"
      isLoading={isLoading}
      skeleton={<ScheduledScanPageLoadingState stableSurfaceRowCount={stableSurfaceRowCount} />}
      className="flex flex-col"
    >
      {pageContent}
    </ContentHandoff>
  )
}
