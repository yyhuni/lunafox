"use client"

import Link from "next/link"
import type { ReactNode } from "react"
import { useLocale, useTranslations } from "next-intl"
import {
  IconChevronRight,
  IconClock,
  semanticIcons,
  type Icon,
} from "@/components/icons"
import { ScanStatusBadge } from "@/components/scan/scan-status-badge"
import { SegmentedMetricProgress } from "@/components/shared/metrics/segmented-metric-progress"
import {
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
  OverviewRuntimeDetailsCard,
  OverviewRuntimeDetailsLayout,
} from "@/components/overview/overview-section-layouts"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { useDatabaseHealth } from "@/hooks/use-database-health"
import { useServerRuntimeMetrics } from "@/hooks/use-overview"
import { useScans, useScanStatistics } from "@/hooks/use-scans"
import { getDateLocale } from "@/lib/date-utils"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type { ScanListRecord } from "@/types/scan.types"

const ScanIcon = semanticIcons.concept.scan
const DatabaseIcon = semanticIcons.concept.database
const ServerIcon = semanticIcons.concept.server
const RECENT_SCAN_PAGE_SIZE = 6
// Server runtime metrics do not expose configurable alert limits; this is an overview-only visual threshold.
const SERVER_RESOURCE_VISUAL_ALERT_THRESHOLD = 80

type RuntimeDetailKey = "scans" | "database" | "resources"
type RuntimeDetail = {
  label: string
  value: string
}

type ScanRuntimeStatusDetail = RuntimeDetail

type RuntimeProgressDetail = RuntimeDetail & {
  progress: number
}

type RuntimeResourceDetail = {
  label: string
  detail: string
  Icon: Icon
  progress: number
}

type RuntimeDetailItemBase = {
  key: RuntimeDetailKey
  label: string
  Icon: Icon
  action?: {
    href: string
    label: string
  }
}

type ScanRuntimeDetailsData = {
  statuses: [
    ScanRuntimeStatusDetail,
    ScanRuntimeStatusDetail,
    ScanRuntimeStatusDetail,
    ScanRuntimeStatusDetail,
    ScanRuntimeStatusDetail,
  ]
  recentScans: {
    status: "loading" | "error" | "empty" | "ready"
    scans: ScanListRecord[]
  }
}

type DatabaseRuntimeDetailsData = {
  status: RuntimeDetail
  connectionUsage: RuntimeProgressDetail
  connectionCount: RuntimeDetail
  diagnostics: [RuntimeDetail, RuntimeDetail]
}

type ResourceRuntimeDetailsData = {
  updatedAt: string
  updatedAtLabel: string
  metrics: [RuntimeResourceDetail, RuntimeResourceDetail, RuntimeResourceDetail]
}

type RuntimeDetailItem =
  | (RuntimeDetailItemBase & { key: "scans"; details: ScanRuntimeDetailsData | null })
  | (RuntimeDetailItemBase & { key: "database"; details: DatabaseRuntimeDetailsData | null })
  | (RuntimeDetailItemBase & { key: "resources"; details: ResourceRuntimeDetailsData | null })

function formatPercent(value: number) {
  return `${Math.round(value)}%`
}

function formatRuntimeDuration(
  totalSeconds: number,
  format: (unit: "seconds" | "minutes" | "hours", count: number) => string,
) {
  if (totalSeconds >= 3600) return format("hours", Math.floor(totalSeconds / 3600))
  if (totalSeconds >= 60) return format("minutes", Math.floor(totalSeconds / 60))
  return format("seconds", Math.max(0, Math.floor(totalSeconds)))
}

function formatRuntimeUpdatedAt(value: string, locale: string) {
  const date = new Date(value)

  if (Number.isNaN(date.getTime())) return null

  return date.toLocaleTimeString(getDateLocale(locale), {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  })
}

function formatScanCreatedAt(value: string, locale: string) {
  const date = new Date(value)

  if (Number.isNaN(date.getTime())) return null

  return date.toLocaleString(getDateLocale(locale), {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  })
}

function formatResourceCapacity(value: number, locale: string, maximumFractionDigits = 1) {
  return new Intl.NumberFormat(getDateLocale(locale), { maximumFractionDigits }).format(value)
}

function getResourceUsedCapacity(percentage: number, capacity: number) {
  return (Math.min(100, Math.max(0, percentage)) / 100) * capacity
}

function RuntimeResourceCardHeader({
  headingId,
  label,
  updatedAt,
  updatedAtLabel,
}: {
  headingId: string
  label: string
  updatedAt: string
  updatedAtLabel: string
}) {
  return (
    <div className={OVERVIEW_RUNTIME_RESOURCE_HEADER_CLASS}>
      <ServerIcon aria-hidden="true" className="size-4 shrink-0 text-muted-foreground" />
      <h3 id={headingId} className={cn("min-w-0 flex-1 truncate", textRole.sectionTitle)}>{label}</h3>
      <time aria-label={updatedAtLabel} className={cn("ml-auto flex shrink-0 items-center gap-1.5 tabular-nums", textRole.caption)}>
        <IconClock aria-hidden="true" className="size-3.5" />
        <span>{updatedAt}</span>
      </time>
    </div>
  )
}

function RuntimeCardHeader({ item, headingId }: { item: RuntimeDetailItem; headingId: string }) {
  if (item.key === "resources") {
    return (
      <RuntimeResourceCardHeader
        headingId={headingId}
        label={item.label}
        updatedAt={item.details?.updatedAt ?? "-"}
        updatedAtLabel={item.details?.updatedAtLabel ?? "-"}
      />
    )
  }

  const { Icon, action } = item

  return (
    <div className={OVERVIEW_RUNTIME_DETAILS_CARD_HEADER_CLASS}>
      <Icon aria-hidden="true" className="size-4 shrink-0 text-muted-foreground" />
      <h3 id={headingId} className={cn("min-w-0 flex-1 truncate", textRole.sectionTitle)}>{item.label}</h3>
      {action && item.key === "scans" ? (
        <Button
          size="sm"
          variant="link"
          className="h-auto self-start px-0"
          render={<Link href={action.href} />}
        >
          {action.label}
          <IconChevronRight aria-hidden="true" />
        </Button>
      ) : action ? (
        <Tooltip>
          <TooltipTrigger
            render={(
              <Button
                variant="ghost"
                size="icon-sm"
                className="text-muted-foreground hover:text-foreground"
                render={<Link href={action.href} aria-label={action.label} />}
              >
                <IconChevronRight aria-hidden="true" />
              </Button>
            )}
          />
          <TooltipContent>{action.label}</TooltipContent>
        </Tooltip>
      ) : null}
    </div>
  )
}

function RuntimeUnavailableState({ unavailableLabel }: { unavailableLabel: string }) {
  return <p className={cn("border-t pt-4", textRole.bodySubtle)}>{unavailableLabel}</p>
}

function RecentScanColumnHeader() {
  const t = useTranslations("overview.operational.runtimeDetails.scans")

  return (
    <div className={OVERVIEW_RUNTIME_SCAN_RECENT_COLUMN_HEADER_CLASS}>
      <div className={OVERVIEW_RUNTIME_SCAN_RECENT_META_CLASS}>
        <span className={textRole.caption}>{t("recentTarget")}</span>
        <span className={textRole.caption}>{t("recentStatus")}</span>
        <span className={textRole.caption}>{t("recentCreatedAt")}</span>
      </div>
      <span aria-hidden="true" className="size-8 shrink-0" />
    </div>
  )
}

function RecentScanRow({ scan }: { scan: ScanListRecord }) {
  const t = useTranslations("overview.operational.runtimeDetails.scans")
  const locale = useLocale()
  const targetLabel = scan.target?.displayName || scan.target?.name || t("targetUnavailable")
  const createdAtLabel = formatScanCreatedAt(scan.createdAt, locale) ?? t("createdAtUnavailable")
  const statusLabel = t(scan.status)
  const detailsHref = `/scan/history/${scan.id}/overview/`

  return (
    <div className={OVERVIEW_RUNTIME_SCAN_RECENT_ROW_CLASS}>
      <div className={OVERVIEW_RUNTIME_SCAN_RECENT_HEADER_CLASS}>
        <div className={OVERVIEW_RUNTIME_SCAN_RECENT_META_CLASS}>
          <span title={targetLabel} className={cn("truncate", textRole.bodyStrong)}>{targetLabel}</span>
          <ScanStatusBadge status={scan.status} variant="icon-only" label={statusLabel} />
          <span title={createdAtLabel} className={cn("truncate tabular-nums", textRole.bodyStrong)}>{createdAtLabel}</span>
        </div>
        <Tooltip>
          <TooltipTrigger
            render={(
              <Button
                variant="ghost"
                size="icon-sm"
                className="shrink-0 text-muted-foreground hover:text-foreground"
                render={<Link href={detailsHref} aria-label={t("viewRecent")} />}
              >
                <IconChevronRight aria-hidden="true" />
              </Button>
            )}
          />
          <TooltipContent>{t("viewRecent")}</TooltipContent>
        </Tooltip>
      </div>
    </div>
  )
}

function RecentScanDetails({
  recentScans,
}: {
  recentScans: ScanRuntimeDetailsData["recentScans"]
}) {
  const t = useTranslations("overview.operational.runtimeDetails.scans")

  if (recentScans.status !== "ready" || recentScans.scans.length === 0) {
    const message = recentScans.status === "loading"
      ? t("recentLoading")
      : recentScans.status === "error"
        ? t("recentUnavailable")
        : t("noRecent")

    return (
      <div className={OVERVIEW_RUNTIME_SCAN_RECENT_LIST_CLASS}>
        <div className={OVERVIEW_RUNTIME_SCAN_RECENT_CLASS}>
          <p className={textRole.bodySubtle}>{message}</p>
        </div>
      </div>
    )
  }

  return (
    <div className={OVERVIEW_RUNTIME_SCAN_RECENT_LIST_CLASS}>
      <div className={OVERVIEW_RUNTIME_SCAN_RECENT_CLASS}>
        <RecentScanColumnHeader />
        <ScrollArea className="min-h-0 flex-1" contentClassName="min-w-0">
          <div className={OVERVIEW_RUNTIME_SCAN_RECENT_ROWS_CLASS}>
            {recentScans.scans.map((scan) => <RecentScanRow key={scan.id} scan={scan} />)}
          </div>
        </ScrollArea>
      </div>
    </div>
  )
}

function ScanRuntimeDetails({ details }: { details: ScanRuntimeDetailsData }) {
  const { statuses, recentScans } = details

  return (
    <dl className={OVERVIEW_RUNTIME_SCAN_BODY_CLASS}>
      <div className={OVERVIEW_RUNTIME_SCAN_STATUS_GRID_CLASS}>
        {statuses.map((detail) => (
          <div key={detail.label} className={OVERVIEW_RUNTIME_SCAN_STATUS_ITEM_CLASS}>
            <dd className={textRole.metricValueDisplay}>{detail.value}</dd>
            <dt title={detail.label} className={cn("min-w-0 truncate", textRole.caption)}>{detail.label}</dt>
          </div>
        ))}
      </div>
      <RecentScanDetails recentScans={recentScans} />
    </dl>
  )
}

function DatabaseRuntimeDetails({ details }: { details: DatabaseRuntimeDetailsData }) {
  const { status, connectionUsage, connectionCount, diagnostics } = details

  return (
    <dl className={OVERVIEW_RUNTIME_DATABASE_BODY_CLASS}>
      <div className={OVERVIEW_RUNTIME_DATABASE_STATUS_ANCHOR_CLASS}>
        <dt className={textRole.caption}>{status.label}</dt>
        <dd className={cn("min-w-0 truncate text-right", textRole.bodyStrong)}>{status.value}</dd>
      </div>
      <div className={OVERVIEW_RUNTIME_DATABASE_METRIC_GRID_CLASS}>
        <div className={OVERVIEW_RUNTIME_DATABASE_METRIC_CLASS}>
          <dt title={connectionUsage.label} className={cn("truncate", textRole.caption)}>{connectionUsage.label}</dt>
          <dd className={textRole.metricValueDisplay}>{connectionUsage.value}</dd>
        </div>
        <div className={OVERVIEW_RUNTIME_DATABASE_METRIC_CLASS}>
          <dt title={connectionCount.label} className={cn("truncate", textRole.caption)}>{connectionCount.label}</dt>
          <dd title={connectionCount.value} className={cn("truncate", textRole.metricValueDisplay)}>{connectionCount.value}</dd>
        </div>
      </div>
      <Progress
        aria-label={`${connectionUsage.label} ${connectionUsage.value}`}
        value={connectionUsage.progress}
        className={OVERVIEW_RUNTIME_DATABASE_PROGRESS_CLASS}
        indicatorClassName="bg-muted-foreground"
      />
      <div className={OVERVIEW_RUNTIME_DATABASE_DIAGNOSTIC_GRID_CLASS}>
        {diagnostics.map((diagnostic) => (
          <div key={diagnostic.label} className={OVERVIEW_RUNTIME_DATABASE_DIAGNOSTIC_CLASS}>
            <dt title={diagnostic.label} className={cn("truncate", textRole.caption)}>{diagnostic.label}</dt>
            <dd title={diagnostic.value} className={cn("truncate tabular-nums", textRole.bodyStrong)}>{diagnostic.value}</dd>
          </div>
        ))}
      </div>
    </dl>
  )
}

function ResourcesRuntimeDetails({ details }: { details: ResourceRuntimeDetailsData }) {
  return (
    <div className={OVERVIEW_RUNTIME_RESOURCE_LIST_CLASS}>
      {details.metrics.map((detail) => {
        const MetricIcon = detail.Icon

        return (
          <SegmentedMetricProgress
            key={detail.label}
            label={detail.label}
            value={detail.progress}
            icon={<MetricIcon aria-hidden="true" />}
            detail={detail.detail}
            variant="runtime-card"
            threshold={SERVER_RESOURCE_VISUAL_ALERT_THRESHOLD}
            barTone="threshold"
            valueTone="neutral"
            className={OVERVIEW_RUNTIME_RESOURCE_ROW_CLASS}
          />
        )
      })}
    </div>
  )
}

function RuntimeDetailsBody({
  item,
  unavailableLabel,
}: {
  item: RuntimeDetailItem
  unavailableLabel: string
}) {
  if (!item.details) return <RuntimeUnavailableState unavailableLabel={unavailableLabel} />

  switch (item.key) {
    case "scans":
      return <ScanRuntimeDetails details={item.details} />
    case "database":
      return <DatabaseRuntimeDetails details={item.details} />
    case "resources":
      return <ResourcesRuntimeDetails details={item.details} />
  }
}

function RuntimeDetailsCell({
  item,
  unavailableLabel,
}: {
  item: RuntimeDetailItem
  unavailableLabel: string
}) {
  const headingId = `overview-runtime-${item.key}`

  return (
    <section data-runtime-card={item.key} aria-labelledby={headingId} className={OVERVIEW_RUNTIME_DETAILS_CARD_SECTION_CLASS}>
      <OverviewRuntimeDetailsCard contentClassName={item.key === "scans" ? OVERVIEW_RUNTIME_SCAN_CARD_CONTENT_CLASS : undefined}>
        <RuntimeCardHeader item={item} headingId={headingId} />
        <RuntimeDetailsBody
          item={item}
          unavailableLabel={unavailableLabel}
        />
      </OverviewRuntimeDetailsCard>
    </section>
  )
}

export function OverviewScanQueue() {
  const t = useTranslations("overview.operational.runtimeDetails")
  const locale = useLocale()
  const scans = useScanStatistics()
  // Fetch exactly the six rows the fixed compact panel renders while matching
  // scan history's ordering instead of privileging currently running work.
  const recentScans = useScans({ page: 1, pageSize: RECENT_SCAN_PAGE_SIZE, orderBy: "createdAt desc" })
  const item: RuntimeDetailItem = {
    key: "scans",
    label: t("scans.recentTitle"),
    Icon: ScanIcon,
    action: {
      href: "/scan/history/",
      label: t("scans.viewHistory"),
    },
    details: scans.isError || !scans.data
      ? null
      : {
          statuses: [
            {
              label: t("scans.running"),
              value: scans.data.running.toLocaleString(locale),
            },
            {
              label: t("scans.pending"),
              value: scans.data.pending.toLocaleString(locale),
            },
            {
              label: t("scans.failed"),
              value: scans.data.failed.toLocaleString(locale),
            },
            {
              label: t("scans.cancelled"),
              value: scans.data.cancelled.toLocaleString(locale),
            },
            {
              label: t("scans.succeeded"),
              value: scans.data.succeeded.toLocaleString(locale),
            },
          ],
          recentScans: {
            status: recentScans.isError
              ? "error"
              : recentScans.isLoading
                ? "loading"
                : recentScans.data
                  ? recentScans.data.results.length > 0
                    ? "ready"
                    : "empty"
                  : "loading",
            scans: recentScans.data?.results ?? [],
          },
        },
  }

  return <RuntimeDetailsCell item={item} unavailableLabel={t("unavailable")} />
}

export function OverviewRuntimeDetails({ leading }: { leading: ReactNode }) {
  const t = useTranslations("overview.operational.runtimeDetails")
  const tReadiness = useTranslations("overview.operational.readiness")
  const locale = useLocale()
  const database = useDatabaseHealth()
  const resources = useServerRuntimeMetrics()
  const items: RuntimeDetailItem[] = [
    {
      key: "database",
      label: tReadiness("database.label"),
      Icon: DatabaseIcon,
      action: {
        href: "/settings/database-health/",
        label: t("database.viewHealth"),
      },
      details: database.isError || !database.data
        ? null
        : {
            status: {
              label: t("database.status"),
              value: tReadiness(`database.${database.data.status}`),
            },
            connectionUsage: {
              label: t("database.connectionUsage"),
              value: formatPercent(database.data.coreSignals.connectionUsagePercent),
              progress: database.data.coreSignals.connectionUsagePercent,
            },
            connectionCount: {
              label: t("database.connectionCount"),
              value: `${database.data.coreSignals.connectionsUsed.toLocaleString(locale)} / ${database.data.coreSignals.connectionsMax.toLocaleString(locale)}`,
            },
            diagnostics: [
              {
                label: t("database.taskBacklog"),
                value: formatRuntimeDuration(
                  database.data.coreSignals.oldestPendingTaskAgeSec,
                  (unit, count) => t(`database.duration${unit[0].toUpperCase()}${unit.slice(1)}`, { count })
                ),
              },
              {
                label: t("database.probeLatency"),
                value: `${database.data.coreSignals.probeLatencyMs.toLocaleString(locale)} ms`,
              },
            ],
          },
    },
    {
      key: "resources",
      label: tReadiness("resources.label"),
      Icon: ServerIcon,
      details: resources.isError || !resources.data
        ? null
        : (() => {
            const updatedAt = formatRuntimeUpdatedAt(resources.data.latest.updatedAt, locale)
            const updatedAtLabel = updatedAt
              ? t("resources.updatedAt", { time: updatedAt })
              : t("resources.updatedAtUnavailable")

            return {
              updatedAt: updatedAt ?? t("resources.updatedAtUnavailable"),
              updatedAtLabel,
              metrics: [
                {
                  label: t("resources.cpu"),
                  progress: resources.data.latest.cpu,
                  detail: t("resources.cpuCores", {
                    count: formatResourceCapacity(resources.data.capacity.cpuCores, locale),
                  }),
                  Icon: semanticIcons.metric.cpu,
                },
                {
                  label: t("resources.memory"),
                  progress: resources.data.latest.memory,
                  detail: t("resources.memoryUsage", {
                    used: formatResourceCapacity(
                      getResourceUsedCapacity(resources.data.latest.memory, resources.data.capacity.memoryTotalGb),
                      locale
                    ),
                    total: formatResourceCapacity(resources.data.capacity.memoryTotalGb, locale),
                  }),
                  Icon: semanticIcons.metric.memory,
                },
                {
                  label: t("resources.disk"),
                  progress: resources.data.latest.disk,
                  detail: t("resources.diskUsage", {
                    used: formatResourceCapacity(
                      getResourceUsedCapacity(resources.data.latest.disk, resources.data.capacity.diskTotalGb),
                      locale,
                      0
                    ),
                    total: formatResourceCapacity(resources.data.capacity.diskTotalGb, locale, 0),
                  }),
                  Icon: semanticIcons.metric.disk,
                },
              ],
            }
          })(),
    },
  ]

  return (
    <OverviewRuntimeDetailsLayout>
      {leading}
      {items.map((item) => (
        <RuntimeDetailsCell
          key={item.key}
          item={item}
          unavailableLabel={t("unavailable")}
        />
      ))}
    </OverviewRuntimeDetailsLayout>
  )
}
