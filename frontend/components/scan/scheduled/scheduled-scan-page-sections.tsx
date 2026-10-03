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
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card"

import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import { ScheduledScanDataTable } from "./scheduled-scan-data-table"
import { createScheduledScanColumns, type ScheduledScanTranslations } from "./scheduled-scan-columns"
import type { ScheduledScanHorizonBucket, ScheduledScanHorizonWindow, ScheduledScanOverviewUpcoming } from "@/types/scheduled-scan.types"
import {
  SCHEDULED_SCAN_PAGE_SIZE,
  SCHEDULED_SCAN_TABLE_LOADING_ROW_HEIGHT_PX,
  SCHEDULED_SCAN_TIMELINE_BODY_CLASS,
} from "@/components/scan/scheduled/scheduled-scan-page-layout"
import {
  COMPACT_CONTENT_GUTTER_CLASS,
  COMPACT_PAGE_SHELL_CLASS,
} from "@/components/shared/layout/page-shell-density"

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
      className={COMPACT_CONTENT_GUTTER_CLASS}
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
      className={COMPACT_CONTENT_GUTTER_CLASS}
    >
      {children}
    </div>
  )
}

interface ScheduledScanTimelineProps {
  horizonWindow: ScheduledScanHorizonWindow
  horizonBuckets: ScheduledScanHorizonBucket[]
  upcoming: ScheduledScanOverviewUpcoming[]
  asOfTime: string
  labels: {
    title: string
    next24HoursCount: string
    empty: string
    windowLabel: string
    soon: string
    now: string
    future: string
  }
  formatTime: (dateString: string) => string
  formatRelativeTime: (dateString: string) => string
}

const SCHEDULED_SCAN_HORIZON_HOUR_MS = 60 * 60 * 1000

// The horizon axis renders one slot per hour of the server-defined window.
// Every marker binds exclusively to horizonBuckets; the now pointer marks the
// overview snapshot instant inside the window, and the queue cards below
// restate the upcoming list next to their real trigger times.
export function ScheduledScanTimeline({
  horizonWindow,
  horizonBuckets,
  upcoming,
  asOfTime,
  labels,
  formatTime,
  formatRelativeTime,
}: ScheduledScanTimelineProps) {
  const startMs = Date.parse(horizonWindow.start)
  const endMs = Date.parse(horizonWindow.end)
  const nowMs = Date.parse(asOfTime)
  // The service contract rejects malformed timestamps, but a defensive finite
  // check keeps a NaN from leaking into `left: NaN%` inline styles if a future
  // producer violates that contract.
  const windowMs = endMs - startMs
  const windowValid = Number.isFinite(startMs) && Number.isFinite(endMs) && windowMs > 0
  const totalSlots = windowValid ? Math.round(windowMs / SCHEDULED_SCAN_HORIZON_HOUR_MS) : 0

  const bucketsByHour = new Map<number, ScheduledScanHorizonBucket>()
  for (const bucket of horizonBuckets) {
    bucketsByHour.set(Date.parse(bucket.hourStart), bucket)
  }
  // Buckets are UTC-aligned hour starts, but horizonWindow.start carries the
  // snapshot's minute/second remainder. Slots must therefore enumerate the
  // whole hours covered by the window, or every bucket lookup misses.
  const firstSlotMs = Math.ceil(startMs / SCHEDULED_SCAN_HORIZON_HOUR_MS) * SCHEDULED_SCAN_HORIZON_HOUR_MS
  const slots: number[] = []
  for (let slotMs = firstSlotMs; slotMs < endMs; slotMs += SCHEDULED_SCAN_HORIZON_HOUR_MS) {
    slots.push(slotMs)
  }
  // A bucket's occurrences fall anywhere inside its hour, so a bucket whose
  // hour already started may already have fired. Only a bucket starting after
  // the snapshot instant is guaranteed to be the next execution.
  const nextHourMs = slots.find((slotMs) => slotMs > nowMs && bucketsByHour.has(slotMs))
  const nowPercent = windowValid && Number.isFinite(nowMs)
    ? Math.min(100, Math.max(0, ((nowMs - startMs) / windowMs) * 100))
    : 0

  return (
    <ScheduledScanOverviewSectionShell
      title={labels.title}
      headerAside={(
        <span className={cn(textRole.metadataValueStrong, "shrink-0 whitespace-nowrap")}>
          {labels.next24HoursCount}
        </span>
      )}
    >
      {totalSlots === 0 || horizonBuckets.length === 0 ? (
        <div className={cn(SCHEDULED_SCAN_TIMELINE_BODY_CLASS, "mt-3 flex items-center justify-center text-center", textRole.bodySubtle)}>
          {labels.empty}
        </div>
      ) : (
        <div className="mt-4 flex flex-col gap-3">
          <div className="rounded-lg border border-border/70 bg-muted/30 p-3">
            <div className="mb-2 flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
              <span className={cn(textRole.helperText, "shrink-0")}>{labels.windowLabel}</span>
              <div className="flex items-center gap-3">
                <span className="inline-flex items-center gap-1.5">
                  <span className="h-2 w-2 rounded-full bg-success" />
                  <span className={textRole.helperText}>{labels.soon}</span>
                </span>
                <span className="inline-flex items-center gap-1.5">
                  <span className="h-3 w-1 rounded-full bg-warning" />
                  <span className={textRole.helperText}>{labels.now}</span>
                </span>
                <span className="inline-flex items-center gap-1.5">
                  <span className="h-2 w-2 rounded-full bg-primary" />
                  <span className={textRole.helperText}>{labels.future}</span>
                </span>
              </div>
            </div>
            <div className="overflow-x-auto rounded-md border border-border/60 bg-background p-1.5">
            <div className="relative min-w-96">
              <div className="flex h-9 items-center">
                {slots.map((slotMs) => {
                  const bucket = bucketsByHour.get(slotMs)
                  if (!bucket) {
                    return <div key={slotMs} className="h-full flex-1 min-w-0 border-l border-border/50 first:border-l-0" />
                  }
                  const isPast = slotMs + SCHEDULED_SCAN_HORIZON_HOUR_MS <= nowMs
                  const isNext = slotMs === nextHourMs
                  return (
                    <div key={slotMs} className={cn("h-full flex-1 min-w-0 border-l border-border/50 first:border-l-0", isPast && "opacity-45")}>
                      <HoverCard>
                        <HoverCardTrigger
                          render={(
                            <span
                              className="group flex h-full w-full cursor-pointer items-center justify-center"
                              aria-label={formatTime(bucket.hourStart)}
                            >
                              {isNext ? (
                                <span className="inline-flex h-3.5 w-3.5 rounded-full bg-success shadow-md shadow-success/50 ring-2 ring-success/25 transition-transform group-hover:scale-125" />
                              ) : bucket.items.length > 1 ? (
                                <Badge size="compact" variant="secondary" className="bg-primary/10 text-primary transition-transform group-hover:scale-125 hover:bg-primary/10" aria-label={formatTime(bucket.hourStart)}>
                                  {bucket.items.length}
                                </Badge>
                              ) : (
                                <span className={cn(
                                  "h-3.5 w-3.5 rounded-full ring-2 transition-transform group-hover:scale-125",
                                  isPast ? "bg-muted-foreground ring-muted-foreground/15" : "bg-primary ring-primary/20",
                                )} />
                              )}
                            </span>
                          )}
                        />
                        <HoverCardContent side="bottom" align="center">
                          <div className="flex flex-col gap-2">
                            <div className={cn(textRole.helperText, "tabular-nums")}>{formatTime(bucket.hourStart)}</div>
                            <ul className="flex max-h-56 flex-col gap-1.5 overflow-y-auto pr-1">
                              {bucket.items.map((item) => {
                                const ScopeIcon = item.scanMode === "target" ? TargetScopeIcon : OrganizationScopeIcon
                                const scopeLabel = item.scanMode === "target" ? item.targetName : item.organizationName
                                return (
                                  <li key={item.resourceName} className="flex min-w-0 flex-col gap-0.5">
                                    <span className={cn(textRole.bodyStrong, "truncate")}>{item.displayName}</span>
                                    {scopeLabel ? (
                                      <span className="flex items-center gap-1.5">
                                        <ScopeIcon className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                                        <span className={cn(item.scanMode === "target" ? textRole.code : textRole.helperText, "truncate")}>{scopeLabel}</span>
                                      </span>
                                    ) : null}
                                  </li>
                                )
                              })}
                            </ul>
                          </div>
                        </HoverCardContent>
                      </HoverCard>
                    </div>
                  )
                })}
              </div>
              <div className="flex pb-0.5">
                {slots.map((slotMs, index) => (
                  <div key={slotMs} className="flex-1 min-w-0">
                    {index % 3 === 0 ? (
                      <span className={cn(textRole.helperText, "block truncate tabular-nums opacity-70")}>
                        {formatTime(new Date(slotMs).toISOString())}
                      </span>
                    ) : null}
                  </div>
                ))}
              </div>
              <div
                className="pointer-events-none absolute top-0 bottom-0 z-10 w-0.5 bg-warning shadow-lg shadow-warning/60"
                style={{ left: `${nowPercent}%` }}
              >
              </div>
            </div>
            </div>
          </div>
          {upcoming.length > 0 ? (
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-5">
              {upcoming.map((item, index) => {
                const ScopeIcon = item.scanMode === "target" ? TargetScopeIcon : OrganizationScopeIcon
                const scopeLabel = item.scanMode === "target" ? item.targetName : item.organizationName
                const isNextCard = index === 0
                return (
                  <div
                    key={item.resourceName}
                    className={cn(
                      "flex min-w-0 flex-col gap-1.5 rounded-lg border bg-card p-2.5",
                      isNextCard ? "border-success/30" : "border-border/70",
                    )}
                  >
                    <div className="flex items-center justify-between gap-2">
                      {isNextCard ? (
                        <span className="h-2 w-2 shrink-0 rounded-full bg-success" />
                      ) : (
                        <span className={cn(textRole.helperText, "font-mono tabular-nums")}>{index + 1}</span>
                      )}
                      <span className={cn(textRole.metadataValue, "shrink-0 tabular-nums")}>{formatTime(item.nextRunTime)}</span>
                    </div>
                    <div className="flex min-w-0 items-center gap-1.5">
                      <ScopeIcon className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                      <span className={cn(textRole.bodyStrong, "truncate")}>{item.displayName}</span>
                    </div>
                    <div className="flex items-center justify-between gap-2">
                      <span className={cn(item.scanMode === "target" ? textRole.code : textRole.helperText, "min-w-0 truncate")}>
                        {scopeLabel ?? "-"}
                      </span>
                      <span className={cn(textRole.helperText, "shrink-0 whitespace-nowrap", isNextCard && "text-success")}>
                        {formatRelativeTime(item.nextRunTime)}
                      </span>
                    </div>
                  </div>
                )
              })}
            </div>
          ) : null}
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
      lastFailure: tColumns("scheduledScan.lastFailure"),
      viewHistory: tColumns("scheduledScan.viewHistory"),
      viewHistoryHint: tColumns("scheduledScan.viewHistoryHint"),
      hasFailureIndicator: tColumns("scheduledScan.hasFailureIndicator"),
      failureCauses: {
        WORKFLOW_UNAVAILABLE: tColumns("scheduledScan.failureCauses.WORKFLOW_UNAVAILABLE"),
        AGENT_NOT_FOUND: tColumns("scheduledScan.failureCauses.AGENT_NOT_FOUND"),
        CONFIG_RESOURCE_UNAVAILABLE: tColumns("scheduledScan.failureCauses.CONFIG_RESOURCE_UNAVAILABLE"),
        ENGINE_UNAVAILABLE: tColumns("scheduledScan.failureCauses.ENGINE_UNAVAILABLE"),
        TARGET_UNAVAILABLE: tColumns("scheduledScan.failureCauses.TARGET_UNAVAILABLE"),
        INTERNAL_UNAVAILABLE: tColumns("scheduledScan.failureCauses.INTERNAL_UNAVAILABLE"),
      },
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
      inTimeZone: tScan.raw("cron.inTimeZone") as string,
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
    <div className={COMPACT_PAGE_SHELL_CLASS}>
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
