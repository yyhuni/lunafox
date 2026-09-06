"use client"

import React from "react"
import { useTranslations } from "next-intl"

import { PageHeader } from "@/components/common/page-header"
import {
  semanticIcons,
} from "@/components/icons"
import { getDataTableSkeletonRowCount } from "@/components/shared/loading/data-table-skeleton"
import {
  getLoadingOwnerAttributes,
  getLoadingStructureSlotAttributes,
} from "@/components/shared/loading/loading-owner"
import { Badge } from "@/components/ui/badge"

import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import { ScheduledScanDataTable } from "./scheduled-scan-data-table"
import { createScheduledScanColumns, type ScheduledScanTranslations } from "./scheduled-scan-columns"
import type { ScheduledScanOverviewUpcoming } from "@/types/scheduled-scan.types"
import {
  SCHEDULED_SCAN_PAGE_SIZE,
  SCHEDULED_SCAN_TABLE_LOADING_ROW_HEIGHT_PX,
  SCHEDULED_SCAN_TIMELINE_BODY_CLASS,
} from "@/components/scan/scheduled/scheduled-scan-page-layout"

const TargetScopeIcon = semanticIcons.concept.target
const OrganizationScopeIcon = semanticIcons.concept.organization

export function ScheduledScanPageHeader({
  title,
  description,
}: {
  title: string
  description: string
}) {
  return (
    <header {...getLoadingStructureSlotAttributes("scheduled-scan-header")}>
      <PageHeader
        code="SCH-01"
        title={title}
        description={description}
        density="compact"
      />
    </header>
  )
}

export function ScheduledScanInsightGrid({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div
      {...getLoadingStructureSlotAttributes("scheduled-scan-insight-grid")}
      className="px-4 lg:px-6"
    >
      {children}
    </div>
  )
}

interface ScheduledScanOverviewSectionShellProps {
  title: React.ReactNode
  headerAside?: React.ReactNode
  children: React.ReactNode
  className?: string
}

export function ScheduledScanOverviewSectionShell({
  title,
  headerAside,
  children,
  className,
}: ScheduledScanOverviewSectionShellProps) {
  return (
    <section className={cn("border rounded-md bg-card px-4 py-3", className)}>
      <div className="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1">
        <h2 className={textRole.panelTitle}>{title}</h2>
        {headerAside}
      </div>
      {children}
    </section>
  )
}

export function ScheduledScanTableSectionShell({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div
      {...getLoadingStructureSlotAttributes("scheduled-scan-table")}
      className="px-4 pt-2 lg:px-6"
    >
      {children}
    </div>
  )
}

interface ScheduledScanTimelineProps {
  items: ScheduledScanOverviewUpcoming[]
  labels: {
    title: string
    next24HoursCount: string
    soon: string
    empty: string
  }
  formatTime: (dateString: string) => string
  formatDate: (dateString: string) => string
}

export function ScheduledScanTimeline({
  items,
  labels,
  formatTime,
  formatDate,
}: ScheduledScanTimelineProps) {
  const desktopColumnClassName = [
    "xl:grid-cols-5",
    "xl:grid-cols-1",
    "xl:grid-cols-2",
    "xl:grid-cols-3",
    "xl:grid-cols-4",
    "xl:grid-cols-5",
  ][items.length] ?? "xl:grid-cols-5"
  const connectorClassName = cn(
    "bg-border bottom-0 absolute top-0 w-px xl:bottom-auto xl:h-px xl:top-1/2",
    items.length > 1 ? "xl:left-0 xl:w-full" : "xl:hidden"
  )

  return (
    <ScheduledScanOverviewSectionShell
      title={labels.title}
      headerAside={(
        <span className={cn(textRole.metadataValueStrong, "shrink-0 whitespace-nowrap")}>
          {labels.next24HoursCount}
        </span>
      )}
    >
      {items.length === 0 ? (
        <div className={cn(SCHEDULED_SCAN_TIMELINE_BODY_CLASS, "mt-3 flex items-center justify-center text-center", textRole.bodySubtle)}>
          {labels.empty}
        </div>
      ) : (
        <div className={cn(SCHEDULED_SCAN_TIMELINE_BODY_CLASS, "mt-3 space-y-0 xl:grid xl:gap-0 xl:space-y-0", desktopColumnClassName)}>
          {items.map((scan, index) => {
            const nextRunTime = scan.nextRunTime ?? ""
            const dateLabel = nextRunTime ? formatDate(nextRunTime) : ""
            const previousNextRunTime = items[index - 1]?.nextRunTime
            const previousDateLabel = previousNextRunTime
              ? formatDate(previousNextRunTime)
              : ""
            const showDate = index === 0 || dateLabel !== previousDateLabel
            const targetLabel = scan.scanMode === "target" ? scan.targetName : scan.organizationName
            const ScopeIcon = scan.scanMode === "target" ? TargetScopeIcon : OrganizationScopeIcon
            return (
				<div key={scan.resourceName} className="grid grid-cols-[6.5rem_1.5rem_minmax(0,1fr)] min-h-[48px] sm:grid-cols-[10rem_1.5rem_minmax(0,1fr)] xl:grid xl:min-h-0 xl:min-w-0 xl:grid-cols-1 xl:grid-rows-[4.5rem_2rem_minmax(0,1fr)]">
                <div className="pt-0.5 xl:flex xl:min-w-0 xl:flex-col xl:items-center xl:justify-end xl:overflow-hidden xl:px-3 xl:text-center">
                  <div className="flex flex-col gap-0.5 items-start xl:min-w-0 xl:items-center xl:gap-1">
                    <span className={cn(textRole.metadataValue, "shrink-0 whitespace-nowrap")}>
                      {nextRunTime ? formatTime(nextRunTime) : "-"}
                    </span>
                    {index === 0 ? (
                      <Badge
                        variant="secondary"
                        className="bg-info/10 px-2 py-0 text-info hover:bg-info/10"
                      >
                        {labels.soon}
                      </Badge>
                    ) : null}
                  </div>
                  {showDate ? (
                    <div className={cn(textRole.helperText, "max-w-full truncate")}>{dateLabel}</div>
                  ) : null}
                </div>
                <div className="flex justify-center relative xl:h-8 xl:items-center">
                  <div className={connectorClassName} />
                  <span className="bg-card border border-border h-2.5 mt-1.5 rounded-full w-2.5 z-10 xl:mt-0" />
                </div>
                <div className="min-w-0 pb-2 xl:overflow-hidden xl:px-3 xl:pb-0 xl:text-center">
                  <div className="flex flex-wrap gap-2 items-center xl:justify-center">
                    <span className={cn(textRole.bodyStrong, "max-w-full min-w-0 truncate")}>
						{scan.displayName}
                    </span>
                  </div>
                  {targetLabel ? (
                    <div className="flex gap-1.5 items-center max-w-full mt-1 xl:justify-center">
                      <ScopeIcon className="h-3.5 shrink-0 text-muted-foreground w-3.5" />
                      <span className={cn(scan.scanMode === "target" ? textRole.code : textRole.helperText, "min-w-0 truncate")}>
                        {targetLabel}
                      </span>
                    </div>
                  ) : null}
                </div>
              </div>
            )
          })}
        </div>
      )}
    </ScheduledScanOverviewSectionShell>
  )
}

export function ScheduledScanOverviewFailure({
	title,
	description,
	retryLabel,
	onRetry,
}: {
	title: string
	description: string
	retryLabel: string
	onRetry: () => void
}) {
	return (
		<ScheduledScanInsightGrid>
			<ScheduledScanOverviewSectionShell title={title}>
				<div className="flex flex-wrap items-center justify-between gap-3 pt-3">
					<p className={cn("max-w-2xl", textRole.bodySubtle)}>{description}</p>
					<Button type="button" size="sm" variant="outline" onClick={onRetry}>
						{retryLabel}
					</Button>
				</div>
			</ScheduledScanOverviewSectionShell>
		</ScheduledScanInsightGrid>
	)
}

export function ScheduledScanOverviewStaleNotice({
	description,
	retryLabel,
	onRetry,
}: {
	description: string
	retryLabel: string
	onRetry: () => void
}) {
	return (
		<div role="status" className="mb-3 flex flex-wrap items-center justify-between gap-3 border border-warning/30 bg-warning/10 px-3 py-2">
			<p className={textRole.bodySubtle}>{description}</p>
			<Button type="button" size="sm" variant="outline" onClick={onRetry}>
				{retryLabel}
			</Button>
		</div>
	)
}

function ScheduledScanOverviewCardBodyLoadingState({ className }: { className: string }) {
  return (
    <div aria-hidden="true" className={cn("mt-3 rounded-md bg-muted/30 p-4", className)}>
      <div className="flex h-full flex-col justify-between">
        <div className="space-y-3">
          <Skeleton className="h-4 w-2/3 rounded-full" />
          <Skeleton className="h-4 w-1/2 rounded-full" />
          <Skeleton className="h-4 w-3/5 rounded-full" />
        </div>
        <div className="space-y-2">
          <Skeleton className="h-2 w-full rounded-full" />
          <Skeleton className="h-2 w-5/6 rounded-full" />
        </div>
      </div>
    </div>
  )
}

function ScheduledScanTimelineLoadingState() {
  const tScan = useTranslations("scan")

  return (
    <ScheduledScanOverviewSectionShell title={tScan("scheduled.workbench.timeline.title")}>
      <ScheduledScanOverviewCardBodyLoadingState className={SCHEDULED_SCAN_TIMELINE_BODY_CLASS} />
    </ScheduledScanOverviewSectionShell>
  )
}

export function ScheduledScanOverviewLoadingState() {
	return (
		<ScheduledScanInsightGrid>
			<ScheduledScanTimelineLoadingState />
		</ScheduledScanInsightGrid>
	)
}

export function ScheduledScanDataTableLoadingState({
  owner,
  rows,
}: {
  owner?: string
  rows: number
}) {
  if (owner !== undefined && !owner.trim()) {
    throw new Error("ScheduledScanDataTableLoadingState requires a non-empty owner.")
  }

  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const tScan = useTranslations("scan")
  const scheduledScanLoadingTranslations = {
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
  } satisfies ScheduledScanTranslations
  const columns = createScheduledScanColumns({
    formatDate: (value) => value,
    handleEdit: () => {},
    handleDelete: () => {},
    handleToggleStatus: () => {},
    t: scheduledScanLoadingTranslations,
  })

  return (
    <div
      {...(owner ? getLoadingOwnerAttributes({ owner, layer: "workspace", intent: "data" }) : {})}
      data-slot="scheduled-scan-data-table-loading-state"
      className="w-full"
    >
      <ScheduledScanDataTable
        data={[]}
        columns={columns}
        searchPlaceholder={tScan("scheduled.searchPlaceholder")}
        searchValue=""
        page={1}
        pageSize={SCHEDULED_SCAN_PAGE_SIZE}
        total={0}
        cursorPaginationSummary={{ total: 0 }}
        paginationNavigation={{ mode: "cursor", canFirstPage: false, canPreviousPage: false, canNextPage: false }}
        quickFilter="all"
        quickFilterLabels={{
          all: tCommon("actions.all"),
          enabled: tScan("scheduled.workbench.filters.enabled"),
          paused: tScan("scheduled.workbench.filters.paused"),
        }}
        quickFilterCounts={{ all: 0, enabled: 0, paused: 0 }}
        loading
        initialLoading
        loadingRowCount={rows}
        loadingRowHeightEstimate={SCHEDULED_SCAN_TABLE_LOADING_ROW_HEIGHT_PX}
        stableSurfaceRowCount={rows}
      />
    </div>
  )
}

export function ScheduledScanPageLoadingState({
  stableSurfaceRowCount = getDataTableSkeletonRowCount(SCHEDULED_SCAN_PAGE_SIZE),
}: {
  stableSurfaceRowCount?: number
} = {}) {
  const tScan = useTranslations("scan")

  return (
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <ScheduledScanPageHeader
        title={tScan("scheduled.title")}
        description={tScan("scheduled.description")}
      />

      <ScheduledScanInsightGrid>
        <ScheduledScanTimelineLoadingState />
      </ScheduledScanInsightGrid>

      <ScheduledScanTableSectionShell>
        <ScheduledScanDataTableLoadingState rows={stableSurfaceRowCount} />
      </ScheduledScanTableSectionShell>
    </div>
  )
}
