import type { ComponentProps, ReactNode } from "react"

import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

export const OVERVIEW_TOP_GRID_CLASS = "grid grid-cols-1 items-stretch gap-4 px-4 lg:px-6 xl:grid-cols-[minmax(0,1.35fr)_minmax(340px,0.65fr)]"
export const OVERVIEW_MIDDLE_GRID_CLASS = "grid grid-cols-1 items-stretch gap-4 px-4 lg:px-6 lg:grid-cols-[minmax(0,1.35fr)_minmax(340px,0.65fr)]"
export const OVERVIEW_SINGLE_SECTION_CLASS = "px-4 py-6 lg:px-6"
export const OVERVIEW_SECTIONS_SHELL_CLASS = "divide-y divide-border border-y border-border"
export const OVERVIEW_RUNTIME_REGION_LOADING_SLOT = "overview-runtime-region"
export const OVERVIEW_ASSET_REGION_LOADING_SLOT = "overview-asset-region"
export const OVERVIEW_OPERATIONAL_REGION_LOADING_SLOT = "overview-operational-region"
export const OVERVIEW_LATEST_CRITICAL_TABLE_CLASS = "min-w-[520px]"
export const OVERVIEW_CURRENT_TASKS_TABLE_CLASS = "min-w-[620px]"
export const OVERVIEW_ASSET_TOP_METRICS_CLASS = "grid grid-cols-1 gap-4 border-b pb-4 sm:grid-cols-2"
export const OVERVIEW_ASSET_TOP_METRIC_ITEM_CLASS = "border-b pb-3 last:border-b-0 last:pb-0 sm:border-b-0 sm:border-r sm:pb-0 sm:pr-4 sm:last:border-r-0"
export const OVERVIEW_ASSET_METRIC_LABEL_ROW_CLASS = "flex items-center gap-3"
export const OVERVIEW_ASSET_DISTRIBUTION_HEADER_GRID_CLASS = "grid grid-cols-[minmax(0,1fr)_96px_96px] gap-3 border-b px-3 py-2"
export const OVERVIEW_ASSET_DISTRIBUTION_ROW_CLASS = "grid grid-cols-[minmax(0,1fr)_96px_96px] items-center gap-3 px-3 py-3"
export const OVERVIEW_TREND_HEADER_CLASS = "flex flex-wrap items-start justify-between gap-4"
export const OVERVIEW_RISK_RING_LEGEND_CLASS = "flex min-w-0 flex-col items-center gap-5 sm:flex-row"
export const OVERVIEW_RISK_RING_CLASS = "relative size-44 shrink-0"
export const OVERVIEW_RISK_LEGEND_CLASS = "w-full min-w-0 flex-1 space-y-3"
export const OVERVIEW_RISK_LEGEND_ROW_CLASS = "flex w-full min-w-0 items-center justify-between gap-4 px-0 py-1"
export const OVERVIEW_RISK_TREND_CHART_CLASS = "h-24 min-w-0"
export const OVERVIEW_SERVER_RESOURCE_CHART_CLASS = "overview-chart-enter h-full min-h-24 min-w-0"
export const OVERVIEW_SERVER_RESOURCE_METRIC_VALUE_ROW_CLASS =
  "flex w-full min-w-0 flex-col gap-1 @2xl/panel:flex-row @2xl/panel:items-baseline @2xl/panel:justify-between @2xl/panel:gap-2"
export const OVERVIEW_SCAN_SUMMARY_GRID_CLASS = "grid grid-cols-1 divide-y divide-border border-y border-border text-foreground sm:grid-cols-5 sm:divide-x sm:divide-y-0"
export const OVERVIEW_SCAN_SUMMARY_ITEM_CLASS = "flex min-h-20 flex-col justify-center px-3 py-3"
export const OVERVIEW_CURRENT_TASKS_HEADER_CLASS = "mb-2 flex items-center justify-between gap-3"
export const OVERVIEW_CURRENT_TASKS_TABLE_VIEWPORT_CLASS = "overflow-x-auto border-t"
export const OVERVIEW_OPERATIONAL_GRID_CLASS = "grid grid-cols-1 items-stretch gap-0 px-4 py-6 lg:px-6 xl:grid-cols-2 [&>section+section]:border-t [&>section+section]:border-border xl:[&>section+section]:border-l xl:[&>section+section]:border-t-0 xl:[&>section+section]:pl-8 xl:[&>section:first-child]:pr-8 [&>section]:min-w-0"
export const OVERVIEW_AGENT_LOCATION_MAP_HEADER_CLASS = "flex min-w-0 flex-wrap items-center justify-between gap-x-4 gap-y-2"
export const OVERVIEW_AGENT_LOCATION_MAP_BODY_CLASS = "flex min-w-0 flex-col pt-3 xl:min-h-0 xl:flex-1"
// Only the primary visualizations gain height on wide canvases; compact metrics keep their established rhythm.
export const OVERVIEW_AGENT_LOCATION_MAP_SURFACE_CLASS = "relative h-full min-h-0 w-full @7xl/main:h-80"
export const OVERVIEW_AGENT_LOCATION_MAP_STATUS_LEGEND_CLASS = "pointer-events-none absolute bottom-2 right-2 z-10 flex max-w-[calc(100%-1rem)] flex-wrap items-center justify-end gap-x-3 gap-y-1 text-foreground"
export const OVERVIEW_ACTION_ITEM_CLASS = "flex min-w-0 items-center gap-3 border-b py-4 last:border-b-0 first:pt-0 last:pb-0"
export const OVERVIEW_RISK_SUMMARY_GRID_CLASS = "grid min-w-0 gap-5 sm:grid-cols-[minmax(0,1.1fr)_minmax(180px,0.9fr)]"
export const OVERVIEW_RISK_SEVERITY_LIST_CLASS = "grid gap-2"
export const OVERVIEW_RISK_SEVERITY_ROW_CLASS = "flex min-w-0 items-center justify-between gap-3 py-1"
export const OVERVIEW_RISK_SUMMARY_METRICS_CLASS = "flex min-w-0 flex-col gap-4 border-t pt-5 sm:border-l sm:border-t-0 sm:pl-5 sm:pt-0"
export const OVERVIEW_RISK_SUMMARY_DETAILS_CLASS = "grid gap-2"
export const OVERVIEW_RISK_SUMMARY_DETAIL_ROW_CLASS = "flex min-w-0 items-center justify-between gap-3"
export const OVERVIEW_ASSET_OVERVIEW_GRID_CLASS = "grid min-w-0 gap-0 xl:grid-cols-[minmax(280px,0.8fr)_minmax(0,1.2fr)] xl:[&>div:first-child]:pr-6"
export const OVERVIEW_ASSET_METRICS_GRID_CLASS = "grid grid-cols-2 divide-x border-b"
export const OVERVIEW_ASSET_METRIC_CLASS = "flex min-w-0 flex-col gap-2 px-4 py-4 first:pl-0 last:pr-0"
// Keep the four distribution rows spread through the left column's remaining visualization slot.
export const OVERVIEW_ASSET_DISTRIBUTION_CHART_CLASS = "aspect-auto h-56 min-h-0 w-full"
export const OVERVIEW_ASSET_CHART_SHELL_CLASS = "h-56 min-w-0 @7xl/main:h-80"
export const OVERVIEW_RUNTIME_DETAILS_SECTION_CLASS = "min-w-0"
export const OVERVIEW_RUNTIME_DETAILS_GRID_CLASS =
  "grid min-w-0 grid-cols-1 gap-0 xl:grid-cols-[minmax(0,1.5fr)_minmax(0,0.75fr)_minmax(0,0.75fr)]"
export const OVERVIEW_RUNTIME_DETAILS_CARD_SECTION_CLASS = "h-full min-w-0 border-t border-border pt-6 first:border-t-0 first:pt-0 xl:border-l xl:border-t-0 xl:pt-0 xl:pl-8 xl:pr-8 xl:first:border-l-0 xl:first:pl-0 xl:last:pr-0"
export const OVERVIEW_RUNTIME_DETAILS_ITEM_CLASS = "h-full min-w-0"
export const OVERVIEW_RUNTIME_DETAILS_ITEM_CONTENT_CLASS = "flex h-full min-h-52 flex-col gap-4"
export const OVERVIEW_RUNTIME_SCAN_CARD_CONTENT_CLASS = "gap-2"
export const OVERVIEW_RUNTIME_DETAILS_CARD_HEADER_CLASS = "flex min-w-0 items-center gap-3"
export const OVERVIEW_RUNTIME_SCAN_BODY_CLASS = "flex min-h-0 min-w-0 flex-1 flex-col"
export const OVERVIEW_RUNTIME_SCAN_STATUS_GRID_CLASS = "grid grid-cols-5 divide-x border-b border-border pb-2"
export const OVERVIEW_RUNTIME_SCAN_STATUS_ITEM_CLASS = "flex min-w-0 flex-col items-center gap-1.5 px-2 text-center first:pl-0 last:pr-0 sm:px-3"
export const OVERVIEW_RUNTIME_SCAN_RECENT_LIST_CLASS = "h-[197px] min-w-0 shrink-0"
export const OVERVIEW_RUNTIME_SCAN_RECENT_CLASS = "flex h-full min-h-0 min-w-0 flex-col gap-1"
export const OVERVIEW_RUNTIME_SCAN_RECENT_ROWS_CLASS = "grid min-w-0 gap-0.5"
export const OVERVIEW_RUNTIME_SCAN_RECENT_ROW_CLASS = "grid min-w-0"
export const OVERVIEW_RUNTIME_SCAN_RECENT_HEADER_CLASS = "flex min-w-0 items-center gap-3"
export const OVERVIEW_RUNTIME_SCAN_RECENT_COLUMN_HEADER_CLASS = "flex min-w-0 items-center gap-3 border-b border-border pb-1"
export const OVERVIEW_RUNTIME_SCAN_RECENT_META_CLASS = "grid min-w-0 flex-1 grid-cols-[minmax(0,1fr)_minmax(72px,0.65fr)_minmax(0,0.85fr)] gap-x-2 sm:grid-cols-[minmax(0,1fr)_minmax(72px,0.6fr)_minmax(0,1fr)] sm:gap-x-4"
export const OVERVIEW_RUNTIME_DATABASE_BODY_CLASS = "flex flex-1 flex-col justify-between gap-3"
export const OVERVIEW_RUNTIME_DATABASE_METRIC_GRID_CLASS = "grid grid-cols-2 gap-3"
export const OVERVIEW_RUNTIME_DATABASE_METRIC_CLASS = "min-w-0 space-y-1"
export const OVERVIEW_RUNTIME_DATABASE_STATUS_ANCHOR_CLASS = "flex min-w-0 items-baseline justify-between gap-3 border-b pb-4"
export const OVERVIEW_RUNTIME_DATABASE_PROGRESS_CLASS = "h-2 bg-muted"
export const OVERVIEW_RUNTIME_DATABASE_DIAGNOSTIC_GRID_CLASS = "grid grid-cols-2 gap-3 border-t pt-3"
export const OVERVIEW_RUNTIME_DATABASE_DIAGNOSTIC_CLASS = "min-w-0 space-y-1"
export const OVERVIEW_RUNTIME_RESOURCE_HEADER_CLASS = "flex min-w-0 items-center gap-3"
export const OVERVIEW_RUNTIME_RESOURCE_LIST_CLASS = "flex flex-1 flex-col justify-between"
export const OVERVIEW_RUNTIME_RESOURCE_ROW_CLASS = "min-w-0"

export function OverviewSectionPanel({
  title,
  children,
  className,
  action,
  contentClassName,
}: {
  title?: ReactNode
  children: ReactNode
  className?: string
  action?: ReactNode
  contentClassName?: string
}) {
  return (
    <section data-slot="overview-section-panel" className={cn("min-w-0", className)}>
      {title || action ? (
        <div data-slot="overview-section-header" className="mb-4 flex min-w-0 items-start justify-between gap-3">
          {title ? <div data-slot="overview-section-title" className={textRole.sectionTitle}>{title}</div> : <span />}
          {action ? <div data-slot="overview-section-action" className="shrink-0 self-start">{action}</div> : null}
        </div>
      ) : null}
      <div data-slot="overview-section-content" className={cn("@container/panel min-w-0", contentClassName)}>{children}</div>
    </section>
  )
}

export function OverviewTopSectionGrid({ children }: { children: ReactNode }) {
  return <div className={OVERVIEW_TOP_GRID_CLASS}>{children}</div>
}

export function OverviewMiddleSectionGrid({ children }: { children: ReactNode }) {
  return <div className={OVERVIEW_MIDDLE_GRID_CLASS}>{children}</div>
}

export function OverviewSingleSection({ children, className, ...props }: ComponentProps<"div">) {
  return <div {...props} className={cn(OVERVIEW_SINGLE_SECTION_CLASS, className)}>{children}</div>
}

export function OverviewOperationalGrid({ children, className, ...props }: ComponentProps<"div">) {
  return <div {...props} className={cn(OVERVIEW_OPERATIONAL_GRID_CLASS, className)}>{children}</div>
}

export function OverviewRuntimeDetailsLayout({ children }: { children: ReactNode }) {
  return (
    <div className={OVERVIEW_RUNTIME_DETAILS_SECTION_CLASS}>
      <div className={OVERVIEW_RUNTIME_DETAILS_GRID_CLASS}>{children}</div>
    </div>
  )
}

export function OverviewRuntimeDetailsCard({ children, contentClassName }: { children: ReactNode; contentClassName?: string }) {
  return (
    <div data-slot="overview-runtime-detail" className={OVERVIEW_RUNTIME_DETAILS_ITEM_CLASS}>
      <div data-slot="overview-runtime-detail-content" className={cn(OVERVIEW_RUNTIME_DETAILS_ITEM_CONTENT_CLASS, contentClassName)}>{children}</div>
    </div>
  )
}

export function OverviewAssetStatusLayout({
  title,
  className,
  topMetrics,
  distribution,
  trendHeader,
  trendChart,
}: {
  title: ReactNode
  className?: string
  topMetrics: ReactNode
  distribution: ReactNode
  trendHeader: ReactNode
  trendChart: ReactNode
}) {
  return (
    <OverviewSectionPanel title={title} className={className}>
      <div className="grid min-w-0 gap-6 @2xl/panel:grid-cols-[minmax(280px,0.8fr)_minmax(420px,1.2fr)]">
        <div className="grid min-w-0 gap-4">
          {topMetrics}
          {distribution}
        </div>

        <div className="flex min-w-0 flex-col border-t pt-4 @2xl/panel:border-l @2xl/panel:border-t-0 @2xl/panel:pl-6 @2xl/panel:pt-0">
          {trendHeader}
          {trendChart}
        </div>
      </div>
    </OverviewSectionPanel>
  )
}

export function OverviewRiskStatusLayout({
  title,
  className,
  ringAndLegend,
  latestCriticalTable,
}: {
  title: ReactNode
  className?: string
  ringAndLegend: ReactNode
  latestCriticalTable: ReactNode
}) {
  return (
    <OverviewSectionPanel title={title} className={className}>
      <div className="grid min-w-0 gap-6 @2xl/panel:grid-cols-[minmax(220px,280px)_minmax(0,1fr)]">
        {ringAndLegend}

        <div className="flex min-w-0 flex-col border-t pt-4 @2xl/panel:border-l @2xl/panel:border-t-0 @2xl/panel:pl-6 @2xl/panel:pt-0">
          {latestCriticalTable}
        </div>
      </div>
    </OverviewSectionPanel>
  )
}

export function OverviewServerResourceUsageLayout({
  title,
  action,
  className,
  contentClassName,
  metricSelector,
  chart,
}: {
  title: ReactNode
  action?: ReactNode
  className?: string
  contentClassName?: string
  metricSelector: ReactNode
  chart: ReactNode
}) {
  return (
    <OverviewSectionPanel
      title={title}
      action={action}
      className={className}
      contentClassName={contentClassName}
    >
      <div className="flex h-full min-h-0 flex-col gap-4">
        {metricSelector}
        <div className="min-h-0 flex-1">
          {chart}
        </div>
      </div>
    </OverviewSectionPanel>
  )
}

export function OverviewScanStatusLayout({
  title,
  className,
  summary,
  currentTasks,
}: {
  title: ReactNode
  className?: string
  summary: ReactNode
  currentTasks: ReactNode
}) {
  return (
    <OverviewSectionPanel title={title} className={className}>
      <div className="grid min-w-0 gap-3">
        <div className="min-w-0">
          {summary}
        </div>
        <div className="grid min-w-0 gap-3 pt-1">
          {currentTasks}
        </div>
      </div>
    </OverviewSectionPanel>
  )
}
