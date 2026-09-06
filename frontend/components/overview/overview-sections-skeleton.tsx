"use client"

import { useTranslations } from "next-intl"

import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { Skeleton } from "@/components/ui/skeleton"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  OVERVIEW_ASSET_REGION_LOADING_SLOT,
  OVERVIEW_ASSET_CHART_SHELL_CLASS,
  OVERVIEW_ASSET_DISTRIBUTION_CHART_CLASS,
  OVERVIEW_ASSET_METRIC_CLASS,
  OVERVIEW_ASSET_METRICS_GRID_CLASS,
  OVERVIEW_ASSET_OVERVIEW_GRID_CLASS,
  OVERVIEW_AGENT_LOCATION_MAP_BODY_CLASS,
  OVERVIEW_AGENT_LOCATION_MAP_HEADER_CLASS,
  OVERVIEW_AGENT_LOCATION_MAP_STATUS_LEGEND_CLASS,
  OVERVIEW_AGENT_LOCATION_MAP_SURFACE_CLASS,
  OVERVIEW_RISK_LEGEND_CLASS,
  OVERVIEW_RISK_LEGEND_ROW_CLASS,
  OVERVIEW_RISK_RING_CLASS,
  OVERVIEW_RISK_RING_LEGEND_CLASS,
  OVERVIEW_RISK_TREND_CHART_CLASS,
  OVERVIEW_RISK_SUMMARY_DETAIL_ROW_CLASS,
  OVERVIEW_RISK_SUMMARY_DETAILS_CLASS,
  OVERVIEW_RISK_SUMMARY_GRID_CLASS,
  OVERVIEW_RISK_SUMMARY_METRICS_CLASS,
  OVERVIEW_TREND_HEADER_CLASS,
  OVERVIEW_OPERATIONAL_REGION_LOADING_SLOT,
  OVERVIEW_RUNTIME_DATABASE_METRIC_CLASS,
  OVERVIEW_RUNTIME_DATABASE_METRIC_GRID_CLASS,
  OVERVIEW_RUNTIME_DATABASE_DIAGNOSTIC_CLASS,
  OVERVIEW_RUNTIME_DATABASE_DIAGNOSTIC_GRID_CLASS,
  OVERVIEW_RUNTIME_DATABASE_BODY_CLASS,
  OVERVIEW_RUNTIME_DATABASE_STATUS_ANCHOR_CLASS,
  OVERVIEW_RUNTIME_DATABASE_PROGRESS_CLASS,
  OVERVIEW_RUNTIME_DETAILS_CARD_SECTION_CLASS,
  OVERVIEW_RUNTIME_DETAILS_CARD_HEADER_CLASS,
  OVERVIEW_RUNTIME_SCAN_CARD_CONTENT_CLASS,
  OVERVIEW_RUNTIME_RESOURCE_HEADER_CLASS,
  OVERVIEW_RUNTIME_RESOURCE_LIST_CLASS,
  OVERVIEW_RUNTIME_RESOURCE_ROW_CLASS,
  OVERVIEW_RUNTIME_SCAN_RECENT_CLASS,
  OVERVIEW_RUNTIME_SCAN_RECENT_COLUMN_HEADER_CLASS,
  OVERVIEW_RUNTIME_SCAN_RECENT_LIST_CLASS,
  OVERVIEW_RUNTIME_SCAN_RECENT_HEADER_CLASS,
  OVERVIEW_RUNTIME_SCAN_RECENT_META_CLASS,
  OVERVIEW_RUNTIME_SCAN_RECENT_ROWS_CLASS,
  OVERVIEW_RUNTIME_SCAN_RECENT_ROW_CLASS,
  OVERVIEW_RUNTIME_SCAN_BODY_CLASS,
  OVERVIEW_RUNTIME_SCAN_STATUS_GRID_CLASS,
  OVERVIEW_RUNTIME_SCAN_STATUS_ITEM_CLASS,
  OVERVIEW_RUNTIME_REGION_LOADING_SLOT,
  OVERVIEW_SECTIONS_SHELL_CLASS,
  OverviewOperationalGrid,
  OverviewRuntimeDetailsCard,
  OverviewRuntimeDetailsLayout,
  OverviewSectionPanel,
  OverviewSingleSection,
} from "@/components/overview/overview-section-layouts"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

function OverviewTextSkeleton({
  roleClassName,
  className,
  reserveText = "\u00a0",
  inline = false,
  featured = false,
  skeletonClassName,
}: {
  roleClassName: string
  className?: string
  reserveText?: string
  inline?: boolean
  featured?: boolean
  skeletonClassName?: string
}) {
  // Preserve the resolved text role's line box so skeletons cannot alter panel rhythm.
  const Element = inline ? "span" : "div"

  return (
    <Element aria-hidden="true" data-featured={featured ? "true" : undefined} className={cn("relative", roleClassName, className)}>
      <span className="invisible">{reserveText}</span>
      <Skeleton className={cn(inline ? "absolute inset-y-0 left-0" : "absolute inset-0", skeletonClassName)} />
    </Element>
  )
}

function OverviewRiskDetailsActionSkeleton() {
  // Match the resolved text action's line box so the lower regions do not shift after handoff.
  return <ActionSkeleton size="content" widthClassName="w-24" className="h-6" />
}

function OverviewAgentLocationMapSkeleton() {
  return (
    <OverviewSectionPanel
      className="h-full"
      contentClassName="xl:h-full"
    >
      <div className="flex min-w-0 flex-col xl:h-full">
        <div className={OVERVIEW_AGENT_LOCATION_MAP_HEADER_CLASS}>
          <div className="flex min-w-0 flex-wrap items-center gap-x-4 gap-y-1">
            <div className="flex items-center gap-2">
              <Skeleton className="size-4 shrink-0 rounded-none" />
              <OverviewTextSkeleton roleClassName={textRole.caption} className="w-16" />
              <OverviewTextSkeleton roleClassName={textRole.metricValueDisplay} className="w-8" />
            </div>
            <div className="flex items-center gap-1.5">
              <OverviewTextSkeleton roleClassName={textRole.caption} className="w-16" />
              <OverviewTextSkeleton roleClassName={textRole.metadataValueStrong} className="w-10" />
            </div>
            <div className="flex items-center gap-1.5">
              <OverviewTextSkeleton roleClassName={textRole.caption} className="w-14" />
              <OverviewTextSkeleton roleClassName={textRole.metadataValueStrong} className="w-10" />
              <OverviewTextSkeleton roleClassName={textRole.caption} className="w-20" />
            </div>
          </div>
        </div>
        <div className={cn(OVERVIEW_AGENT_LOCATION_MAP_BODY_CLASS, "h-full min-h-0")}>
          <div className={OVERVIEW_AGENT_LOCATION_MAP_SURFACE_CLASS}>
            <Skeleton className="h-full min-h-0 w-full rounded-none opacity-35 aspect-[248/100] xl:aspect-auto" />
            <div
              aria-hidden="true"
              data-slot="agent-status-legend"
              className={OVERVIEW_AGENT_LOCATION_MAP_STATUS_LEGEND_CLASS}
            >
              {Array.from({ length: 4 }).map((_, index) => (
                <div key={index} className="flex items-center gap-1.5">
                  <Skeleton className="size-2 rounded-full" />
                  <OverviewTextSkeleton roleClassName={textRole.caption} className="w-10" />
                  <OverviewTextSkeleton roleClassName={textRole.metadataValueStrong} className="w-5" />
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </OverviewSectionPanel>
  )
}

function OverviewRiskSummarySkeleton({ className }: { className?: string } = {}) {
  const t = useTranslations("overview.operational.risk")

  return (
    <OverviewSectionPanel
      title={(
        <div className="flex min-w-0 items-center gap-2">
          <Skeleton className="size-4 shrink-0 rounded-none" />
          <OverviewTextSkeleton
            roleClassName={textRole.sectionTitle}
            reserveText={t("title")}
            className="min-w-0"
          />
        </div>
      )}
      action={<OverviewRiskDetailsActionSkeleton />}
      className={cn("h-full", className)}
      contentClassName="pt-1"
    >
      <div className={OVERVIEW_RISK_SUMMARY_GRID_CLASS}>
        <div className={OVERVIEW_RISK_RING_LEGEND_CLASS}>
          <div className={OVERVIEW_RISK_RING_CLASS}>
            <Skeleton className="size-full rounded-full" />
            <div className="absolute inset-7 flex flex-col items-center justify-center rounded-full bg-background text-center">
              <OverviewTextSkeleton featured roleClassName={textRole.metricValueDisplay} className="w-12" />
              <OverviewTextSkeleton roleClassName={textRole.badgeSubtle} className="mt-1 w-14" />
            </div>
          </div>
          <div className={OVERVIEW_RISK_LEGEND_CLASS}>
          {Array.from({ length: 5 }).map((_, index) => (
              <div key={index} className={OVERVIEW_RISK_LEGEND_ROW_CLASS}>
                <div className="flex items-center gap-2">
                  <Skeleton className="size-2.5 rounded-full" />
                  <OverviewTextSkeleton roleClassName={textRole.bodySubtle} className="w-16" />
                </div>
                <OverviewTextSkeleton roleClassName={textRole.metadataValueStrong} className="w-10" />
              </div>
            ))}
          </div>
        </div>
        <div className={OVERVIEW_RISK_SUMMARY_METRICS_CLASS}>
          <div className="min-w-0 space-y-4">
            <div className={OVERVIEW_TREND_HEADER_CLASS}>
              <OverviewTextSkeleton roleClassName={textRole.sectionTitle} reserveText={t("trendTitle")} />
            </div>
            <Skeleton className={cn("w-full", OVERVIEW_RISK_TREND_CHART_CLASS)} />
            <div className={OVERVIEW_RISK_SUMMARY_DETAILS_CLASS}>
              {Array.from({ length: 3 }).map((_, index) => (
                <div key={index} className={OVERVIEW_RISK_SUMMARY_DETAIL_ROW_CLASS}>
                  <OverviewTextSkeleton roleClassName={textRole.caption} className="w-20" />
                  <OverviewTextSkeleton
                    roleClassName={index === 0 ? textRole.bodyStrong : textRole.metadataValueStrong}
                    className="w-10"
                  />
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </OverviewSectionPanel>
  )
}

function OverviewAssetOverviewSkeleton() {
  return (
    <OverviewSectionPanel
      contentClassName="pt-1"
    >
      <div className={OVERVIEW_ASSET_OVERVIEW_GRID_CLASS}>
        <div className="space-y-6">
          <div className={OVERVIEW_ASSET_METRICS_GRID_CLASS}>
            {Array.from({ length: 2 }).map((_, index) => (
              <div key={index} className={OVERVIEW_ASSET_METRIC_CLASS}>
                <div className="flex items-center gap-2">
                  <Skeleton className="size-4" />
                  <OverviewTextSkeleton roleClassName={textRole.caption} className="w-20" />
                </div>
                <OverviewTextSkeleton roleClassName={textRole.metricValueDisplay} className="w-24" />
                <OverviewTextSkeleton roleClassName={textRole.caption} className="w-16" />
              </div>
            ))}
          </div>
          <div className={OVERVIEW_ASSET_DISTRIBUTION_CHART_CLASS}>
            <div className="grid h-full min-w-0 content-center gap-3 pl-6">
              {Array.from({ length: 4 }).map((_, index) => (
                <div key={index} className="grid min-w-0 grid-cols-[48px_minmax(0,1fr)_84px] items-center gap-2">
                  <OverviewTextSkeleton roleClassName={textRole.caption} className="w-10" />
                  <Skeleton className="h-5 w-full rounded-sm" />
                  <OverviewTextSkeleton roleClassName={textRole.metadataValue} className="w-20" />
                </div>
              ))}
            </div>
          </div>
        </div>
        <div className="space-y-4 border-t pt-5 xl:border-l xl:border-t-0 xl:pl-6 xl:pt-0">
          <div className="flex items-center justify-between gap-3">
            <OverviewTextSkeleton roleClassName={textRole.sectionTitle} className="w-40" />
            <OverviewTextSkeleton roleClassName={textRole.caption} className="w-8" />
          </div>
          <Skeleton className={cn("w-full", OVERVIEW_ASSET_CHART_SHELL_CLASS)} />
        </div>
      </div>
    </OverviewSectionPanel>
  )
}

function OverviewScanQueueSkeleton() {
  return (
    <section className={OVERVIEW_RUNTIME_DETAILS_CARD_SECTION_CLASS}>
      <OverviewRuntimeDetailsCard contentClassName={OVERVIEW_RUNTIME_SCAN_CARD_CONTENT_CLASS}>
        <div className={OVERVIEW_RUNTIME_DETAILS_CARD_HEADER_CLASS}>
          <Skeleton className="size-4 rounded-none" />
          <OverviewTextSkeleton roleClassName={textRole.sectionTitle} className="w-16" />
          <Skeleton className="ml-auto h-5 w-24 rounded-none" />
        </div>
        <div className={OVERVIEW_RUNTIME_SCAN_BODY_CLASS}>
          <div className={OVERVIEW_RUNTIME_SCAN_STATUS_GRID_CLASS}>
            {Array.from({ length: 5 }).map((_, index) => (
              <div key={index} className={OVERVIEW_RUNTIME_SCAN_STATUS_ITEM_CLASS}>
                <OverviewTextSkeleton roleClassName={textRole.metricValueDisplay} className="w-10" />
                <div className="flex min-w-0 items-center gap-1.5">
                  {index < 2 ? <Skeleton className="size-3.5 shrink-0 rounded-none" /> : null}
                  <OverviewTextSkeleton roleClassName={textRole.caption} className="w-8" />
                </div>
              </div>
            ))}
          </div>
          <div className={OVERVIEW_RUNTIME_SCAN_RECENT_LIST_CLASS}>
            <div className={OVERVIEW_RUNTIME_SCAN_RECENT_CLASS}>
              <div className={OVERVIEW_RUNTIME_SCAN_RECENT_COLUMN_HEADER_CLASS}>
                <div className={OVERVIEW_RUNTIME_SCAN_RECENT_META_CLASS}>
                  <OverviewTextSkeleton roleClassName={textRole.caption} className="w-8" />
                  <OverviewTextSkeleton roleClassName={textRole.caption} className="w-10" />
                  <OverviewTextSkeleton roleClassName={textRole.caption} className="w-16" />
                </div>
                <Skeleton className="size-8 shrink-0 rounded-none" />
              </div>
              <ScrollArea className="min-h-0 flex-1" contentClassName="min-w-0">
                <div className={OVERVIEW_RUNTIME_SCAN_RECENT_ROWS_CLASS}>
                  {Array.from({ length: 6 }).map((_, index) => (
                    <div key={index} className={OVERVIEW_RUNTIME_SCAN_RECENT_ROW_CLASS}>
                      <div className={OVERVIEW_RUNTIME_SCAN_RECENT_HEADER_CLASS}>
                        <div className={OVERVIEW_RUNTIME_SCAN_RECENT_META_CLASS}>
                          <OverviewTextSkeleton roleClassName={textRole.bodyStrong} className="w-24" />
                          <OverviewTextSkeleton roleClassName={textRole.bodyStrong} className="w-10" />
                          <OverviewTextSkeleton roleClassName={textRole.bodyStrong} className="w-28" />
                        </div>
                        <Skeleton className="size-8 shrink-0 rounded-none" />
                      </div>
                    </div>
                  ))}
                </div>
              </ScrollArea>
            </div>
          </div>
        </div>
      </OverviewRuntimeDetailsCard>
    </section>
  )
}

function OverviewRuntimeDetailsSkeleton() {
  return (
    <OverviewRuntimeDetailsLayout>
      <OverviewRiskSummarySkeleton className={OVERVIEW_RUNTIME_DETAILS_CARD_SECTION_CLASS} />

      <section className={OVERVIEW_RUNTIME_DETAILS_CARD_SECTION_CLASS}>
        <OverviewRuntimeDetailsCard>
          <div className={OVERVIEW_RUNTIME_DETAILS_CARD_HEADER_CLASS}>
            <Skeleton className="size-4 rounded-none" />
            <OverviewTextSkeleton roleClassName={textRole.sectionTitle} className="w-14" />
            <Skeleton className="ml-auto size-8 rounded-none" />
          </div>
          <div className={OVERVIEW_RUNTIME_DATABASE_BODY_CLASS}>
            <div className={OVERVIEW_RUNTIME_DATABASE_STATUS_ANCHOR_CLASS}>
              <OverviewTextSkeleton roleClassName={textRole.caption} className="w-10" />
              <OverviewTextSkeleton roleClassName={textRole.metricValueDisplay} className="w-16" />
            </div>
            <div className={OVERVIEW_RUNTIME_DATABASE_METRIC_GRID_CLASS}>
              {Array.from({ length: 2 }).map((_, index) => (
                <div key={index} className={OVERVIEW_RUNTIME_DATABASE_METRIC_CLASS}>
                  <OverviewTextSkeleton roleClassName={textRole.caption} className="w-14" />
                  <OverviewTextSkeleton roleClassName={textRole.metricValueDisplay} className="w-16" />
                </div>
              ))}
            </div>
            <Skeleton className={cn(OVERVIEW_RUNTIME_DATABASE_PROGRESS_CLASS, "w-full")} />
            <div className={OVERVIEW_RUNTIME_DATABASE_DIAGNOSTIC_GRID_CLASS}>
              {Array.from({ length: 2 }).map((_, index) => (
                <div key={index} className={OVERVIEW_RUNTIME_DATABASE_DIAGNOSTIC_CLASS}>
                  <OverviewTextSkeleton roleClassName={textRole.caption} className="w-14" />
                  <OverviewTextSkeleton roleClassName={textRole.bodyStrong} className="w-12" />
                </div>
              ))}
            </div>
          </div>
        </OverviewRuntimeDetailsCard>
      </section>

      <section className={OVERVIEW_RUNTIME_DETAILS_CARD_SECTION_CLASS}>
        <OverviewRuntimeDetailsCard>
          <div className={OVERVIEW_RUNTIME_RESOURCE_HEADER_CLASS}>
            <Skeleton className="size-4 rounded-none" />
            <OverviewTextSkeleton roleClassName={textRole.sectionTitle} className="w-20" />
            <OverviewTextSkeleton roleClassName={textRole.caption} className="ml-auto w-16" />
          </div>
          <div className={OVERVIEW_RUNTIME_RESOURCE_LIST_CLASS}>
            {Array.from({ length: 3 }).map((_, index) => (
              <div key={index} className={cn(OVERVIEW_RUNTIME_RESOURCE_ROW_CLASS, "space-y-1")}>
                <div className="flex items-center justify-between gap-2">
                  <div className="flex items-center gap-2">
                    <Skeleton className="size-3.5 rounded-none" />
                    <OverviewTextSkeleton roleClassName={textRole.caption} className="w-8" />
                  </div>
                  <OverviewTextSkeleton roleClassName={textRole.caption} className="w-16" />
                </div>
                <div className="flex items-center gap-2">
                  <div className="flex h-2 min-w-0 flex-1 gap-0.5">
                    {Array.from({ length: 18 }).map((_, segmentIndex) => (
                      <Skeleton key={segmentIndex} className="min-w-0 flex-1 rounded-none" />
                    ))}
                  </div>
                  <div className="shrink-0 text-right tabular-nums">
                    <OverviewTextSkeleton roleClassName={textRole.metadataValueStrong} inline skeletonClassName="w-8" />
                  </div>
                </div>
              </div>
            ))}
          </div>
        </OverviewRuntimeDetailsCard>
      </section>
    </OverviewRuntimeDetailsLayout>
  )
}

export function OverviewSectionsSkeleton() {
  return (
    <div className={OVERVIEW_SECTIONS_SHELL_CLASS}>
      <OverviewSingleSection data-loading-slot={OVERVIEW_RUNTIME_REGION_LOADING_SLOT}>
        <OverviewRuntimeDetailsSkeleton />
      </OverviewSingleSection>

      <OverviewOperationalGrid data-loading-slot={OVERVIEW_OPERATIONAL_REGION_LOADING_SLOT}>
        <OverviewAgentLocationMapSkeleton />
        <OverviewScanQueueSkeleton />
      </OverviewOperationalGrid>

      <OverviewSingleSection data-loading-slot={OVERVIEW_ASSET_REGION_LOADING_SLOT}>
        <OverviewAssetOverviewSkeleton />
      </OverviewSingleSection>
    </div>
  )
}
