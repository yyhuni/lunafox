"use client"

import React from "react"
import Link from "next/link"
import { useLocale, useTranslations } from "next-intl"

import { Badge } from "@/components/ui/badge"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import {
  DetailDrawer,
  DetailDrawerTabs,
  DetailDrawerTabsContent,
  DetailDrawerTabsList,
  DetailDrawerTabsTrigger,
} from "@/components/shared/detail-drawer"
import { CopyButton } from "@/components/shared/feedback/copy-button"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { getLoadingOwnerAttributes } from "@/components/shared/loading/loading-owner"
import { RawLogViewer } from "@/components/shared/visualization/raw-log-viewer"
import { Skeleton } from "@/components/ui/skeleton"
import {
  AlertTriangle,
  ChevronRight,
  Circle,
  IconBan,
  IconClock,
  IconLoader2,
  semanticIcons,
} from "@/components/icons"
import { ScanLogList } from "@/components/scan/scan-log-list"
import { buildScanLogContent } from "@/components/scan/scan-log-list-state"
import { useScanOverviewState } from "@/components/scan/history/scan-overview-state"
import { useScanExecutedEngineDisplay } from "@/hooks/use-scan-executed-engine-display"
import { useScanWorkflows } from "@/hooks/use-scan-workflows"
import {
  buildTaskProgressLogsByTaskId,
  getRuntimeTaskPresentationStatus,
  getRuntimeTaskDurationSeconds,
  toRuntimeTaskItems,
  type RuntimeTaskItem,
  type RuntimeTaskPresentationStatus,
} from "@/components/scan/history/scan-runtime-detail-utils"
import {
  RUNTIME_DETAIL_PANEL_CLASS,
  RUNTIME_DETAIL_PANEL_FALLBACK_CLASS,
} from "@/components/scan/history/scan-runtime-detail-layout"
import {
  getScanOverviewSummaryItemDividerClass,
  SCAN_OVERVIEW_STICKY_SIDE_PANEL_CLASS,
  SCAN_OVERVIEW_SUMMARY_GRID_CLASS,
  SCAN_OVERVIEW_SUMMARY_ITEM_CLASS,
} from "@/components/scan/history/scan-overview-layout"
import { textRole } from "@/lib/typography"
import { formatLocalTimestampSeconds } from "@/lib/log-time"
import { getAgentHealthDistributionStatus } from "@/lib/agent-status-distribution"
import { getScanWorkflowDisplayName } from "@/lib/scan-workflow-display"
import { serializeWorkflowConfiguration } from "@/lib/workflow-config"
import {
  getScanStatusBadgeVariant,
  getScanStatusClasses,
  getScanStatusSurfaceClass,
  getScanStatusTextClass,
} from "@/lib/status-config"
import type {
  EngineDiagnosticErrorType,
  EngineDiagnosticFailedStage,
  EngineExecutionDiagnostics,
  ResultTypeWatermark,
  ScanLog,
  ScanRecord,
} from "@/types/scan.types"
import { cn } from "@/lib/utils"

type TFn = ReturnType<typeof useTranslations>
type StatusLabelFn = (status: RuntimeTaskItem["status"]) => string
type RuntimePanelTranslationFn = (key: string, params?: Record<string, string | number | Date>) => string
export type RuntimeDetailTab = "logs" | "config"
const RUNTIME_DURATION_REFRESH_MS = 1000

const VulnerabilitiesMetricIcon = semanticIcons.concept.vulnerability

interface ScanRuntimeDetailDrawerProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  scan: ScanRecord | null
}

export function formatDateTime(value?: string, locale: string = "zh") {
  if (!value) return "-"
  try {
    return new Date(value).toLocaleString(locale === "zh" ? "zh-CN" : "en-US", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false,
    })
  } catch {
    return value
  }
}

export function formatTaskStartTime(value?: string) {
  return value ? formatLocalTimestampSeconds(value) : "-"
}

export function formatDurationClock(duration?: number) {
  if (duration === undefined || duration === null || Number.isNaN(duration)) return "--:--:--"

  const totalSeconds = Math.max(0, Math.round(duration))
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60

  return [hours, minutes, seconds].map((value) => String(value).padStart(2, "0")).join(":")
}

function getRuntimeDurationSeconds(scan: ScanRecord, tasks: RuntimeTaskItem[], now: Date = new Date()) {
  if (scan.createdAt && scan.stoppedAt) {
    const started = new Date(scan.createdAt).getTime()
    const stopped = new Date(scan.stoppedAt).getTime()

    if (Number.isFinite(started) && Number.isFinite(stopped) && stopped >= started) {
      return (stopped - started) / 1000
    }
  }

  if (scan.status === "running" && scan.createdAt) {
    const started = new Date(scan.createdAt).getTime()
    const current = now.getTime()

    if (Number.isFinite(started) && Number.isFinite(current) && current >= started) {
      return (current - started) / 1000
    }
  }

  const taskDuration = tasks.reduce((sum, task) => sum + (getRuntimeTaskDurationSeconds(task, now) ?? 0), 0)
  return taskDuration > 0 ? taskDuration : undefined
}

export function hasLiveRuntimeDuration(scan: ScanRecord | null | undefined, tasks: RuntimeTaskItem[]) {
  return Boolean(
    (scan?.status === "running" && scan.createdAt) ||
      tasks.some((task) => task.status === "running" && task.startedAt)
  )
}

export function useRuntimeDurationNow(enabled: boolean) {
  const [now, setNow] = React.useState(() => new Date())

  React.useEffect(() => {
    if (!enabled) return

    setNow(new Date())
    const intervalId = window.setInterval(() => setNow(new Date()), RUNTIME_DURATION_REFRESH_MS)
    return () => window.clearInterval(intervalId)
  }, [enabled])

  return now
}

function statusIcon(status: RuntimeTaskPresentationStatus) {
  const canonicalStatus = status === "interrupted" || status === "not_started" ? "cancelled" : status
  const className = cn("h-4 w-4", getScanStatusTextClass(canonicalStatus))

  switch (status) {
    case "succeeded":
      return <semanticIcons.status.success className={className} />
    case "failed":
      return <semanticIcons.status.failed className={className} />
    case "running":
      return <IconLoader2 className={cn(className, "animate-spin")} />
    case "pending":
      return <IconClock className={className} />
    case "interrupted":
      return <IconBan className={className} />
    case "not_started":
      return <Circle className={className} />
    case "cancelled":
      return <semanticIcons.status.cancelled className={className} />
    case "skipped":
      return <IconBan className={className} />
  }
}

function getRuntimeTaskStatusLabel(
  status: RuntimeTaskPresentationStatus,
  t: TFn,
  statusLabel: StatusLabelFn
) {
  if (status === "interrupted") return t("runtimeDrawer.tasks.status.interrupted")
  if (status === "not_started") return t("runtimeDrawer.tasks.status.notStarted")
  return statusLabel(status)
}

function getRuntimeTaskSkipReasonLabel(task: RuntimeTaskItem, t: TFn) {
  if (task.status !== "skipped") return null
  if (task.skipReason === "user_disabled") return t("runtimeDrawer.tasks.skipReasons.userDisabled")
  if (task.skipReason === "target_not_applicable") return t("runtimeDrawer.tasks.skipReasons.targetNotApplicable")
  return task.skipReason ? t("runtimeDrawer.tasks.skipReasons.other", { reason: task.skipReason }) : t("runtimeDrawer.tasks.skipReasons.planning")
}

function getRuntimeAgentTransportPresentation(scan: ScanRecord, t: TFn) {
  if (scan.agentDeleted) {
    return { label: t("runtimeDrawer.meta.deleted"), variant: "destructive" as const }
  }
  if (scan.agentStatus === "online") {
    return { label: t("runtimeDrawer.meta.online"), variant: "success" as const }
  }
  if (scan.agentStatus === "offline") {
    return { label: t("runtimeDrawer.meta.offline"), variant: "secondary" as const }
  }
  return { label: t("runtimeDrawer.meta.stateUnavailable"), variant: "outline" as const }
}

function getRuntimeAgentHealthPresentation(scan: ScanRecord, t: TFn) {
  if (scan.agentDeleted || !scan.agentHealthState) return null
  const state = getAgentHealthDistributionStatus(scan.agentHealthState)
  if (state === "healthy") {
    return { label: t("runtimeDrawer.meta.healthy"), variant: "success" as const }
  }
  if (state === "warning") {
    return { label: t("runtimeDrawer.meta.warning"), variant: "warning" as const }
  }
  return { label: t("runtimeDrawer.meta.healthUnknown"), variant: "outline" as const }
}

export function RuntimeHeader({
  scan,
  tasks,
  now,
  locale,
  t,
  statusLabel,
}: {
  scan: ScanRecord
  tasks: RuntimeTaskItem[]
  now: Date
  locale: string
  t: TFn
  statusLabel: StatusLabelFn
}) {
  const status = scan.status ?? "pending"
  const { data: scanWorkflows = [] } = useScanWorkflows()
  const workflowDisplayName = React.useMemo(
    () => getScanWorkflowDisplayName(scan.scanWorkflow, scanWorkflows),
    [scan.scanWorkflow, scanWorkflows]
  )
  const agentDisplayName = scan.agentDeleted ? scan.agent : scan.agentName
  const agentTransport = getRuntimeAgentTransportPresentation(scan, t)
  const agentHealth = getRuntimeAgentHealthPresentation(scan, t)

  return (
    <section className="rounded-lg border border-border/60 bg-card px-4 py-4">
      <div className="min-w-0 space-y-3">
        <div className="flex flex-wrap items-center gap-3">
          <h2 className={cn("break-all", textRole.panelTitle)}>
            {scan.target?.displayName || scan.target?.name || t("runtimeDrawer.emptyTarget")}
          </h2>
          <Badge
            variant={getScanStatusBadgeVariant(status)}
            className={cn("gap-1.5 px-2 py-0.5 text-xs", getScanStatusClasses(status))}
          >
            {statusIcon(status)}
            {statusLabel(status)}
          </Badge>
          <Badge variant="outline" className="px-2 py-0.5 text-xs text-muted-foreground">
            #{scan.id}
          </Badge>
        </div>

        <div className="flex flex-wrap items-center gap-x-5 gap-y-2">
          <div className="flex items-center gap-2">
            <span className={cn(textRole.metadataLabel, "shrink-0")}>
              {t("runtimeDrawer.meta.createdAt")}
            </span>
            <span className={textRole.metadataValueStrong}>{formatDateTime(scan.createdAt, locale)}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className={cn(textRole.metadataLabel, "shrink-0")}>
              {t("runtimeDrawer.meta.assignment")}
            </span>
            <span className={textRole.metadataValueStrong}>
              {scan.assignmentMode === "pinned" ? t("runtimeDrawer.meta.pinned") : t("runtimeDrawer.meta.automatic")}
            </span>
          </div>
          {scan.agent && agentDisplayName ? (
            <div className="flex items-center gap-2">
              <span className={cn(textRole.metadataLabel, "shrink-0")}>
                {t("runtimeDrawer.meta.agent")}
              </span>
              <span className={textRole.metadataValueStrong}>{agentDisplayName}</span>
              <Badge variant={agentTransport.variant} className="px-2 py-0.5 text-xs">
                {agentTransport.label}
              </Badge>
              {agentHealth ? (
                <Badge variant={agentHealth.variant} className="px-2 py-0.5 text-xs">
                  {agentHealth.label}
                </Badge>
              ) : null}
            </div>
          ) : null}
          {workflowDisplayName ? (
            <div className="flex min-w-0 items-center gap-2">
              <span className={cn(textRole.metadataLabel, "shrink-0")}>
                {t("runtimeDrawer.meta.workflow")}
              </span>
              <span className={cn(textRole.metadataValueStrong, "truncate")} title={workflowDisplayName}>
                {workflowDisplayName}
              </span>
            </div>
          ) : null}
          <div className="flex items-center gap-2">
            <span className={cn(textRole.metadataLabel, "shrink-0")}>
              {t("inputSource.label")}
            </span>
            <span className={textRole.metadataValueStrong}>
              {t(`inputSource.${scan.inputSource}`)}
            </span>
          </div>
          <div className="flex items-center gap-2">
            <span className={cn(textRole.metadataLabel, "shrink-0")}>
              {t("runtimeDrawer.summary.duration")}
            </span>
            <span className={textRole.metadataValueStrong}>
              {formatDurationClock(getRuntimeDurationSeconds(scan, tasks, now))}
            </span>
          </div>
        </div>
      </div>
    </section>
  )
}

export function RuntimeSummarySection({ scan, tasks, t }: { scan: ScanRecord; tasks: RuntimeTaskItem[]; t: TFn }) {
  const completedCount = tasks.filter((task) => task.status === "succeeded").length
  const stats = scan.cachedStats
  const summaryItems = [
    {
      key: "succeeded",
      label: t("runtimeDrawer.summary.tasks"),
      value: t("runtimeDrawer.summary.taskCount", { completed: completedCount, total: tasks.length }),
    },
    {
      key: "subdomains",
      label: t("cards.subdomains"),
      value: stats?.subdomainsCount ?? 0,
    },
    {
      key: "websites",
      label: t("cards.websites"),
      value: stats?.websitesCount ?? 0,
    },
    {
      key: "ips",
      label: t("cards.ips"),
      value: stats?.ipsCount ?? 0,
    },
    {
      key: "urls",
      label: t("cards.urls"),
      value: stats?.endpointsCount ?? 0,
    },
    {
      key: "risks",
      label: t("runtimeDrawer.summary.risks"),
      value: scan.cachedStats?.vulnsTotal ?? 0,
    },
  ]

  return (
    <section className="space-y-2">
      <div className="overflow-hidden rounded-lg border border-border/60 bg-card">
        <dl className={SCAN_OVERVIEW_SUMMARY_GRID_CLASS}>
          {summaryItems.map((item, index) => (
            <div
              key={item.key}
              className={cn(
                SCAN_OVERVIEW_SUMMARY_ITEM_CLASS,
                getScanOverviewSummaryItemDividerClass(index)
              )}
            >
              <dt className={cn(textRole.metadataLabel, "min-w-0 truncate")}>{item.label}</dt>
              <dd className={cn(textRole.metadataValueStrong, "whitespace-nowrap tabular-nums")}>
                {typeof item.value === "number" ? item.value.toLocaleString() : item.value}
              </dd>
            </div>
          ))}
        </dl>
      </div>
    </section>
  )
}

function RuntimeAssetSummaryPanel({ scan, t }: { scan: ScanRecord; t: TFn }) {
  const stats = scan.cachedStats
  const items = [
    {
      key: "subdomains",
      label: t("cards.subdomains"),
      value: stats?.subdomainsCount ?? 0,
      icon: semanticIcons.concept.subdomain,
      href: `/scan/history/${scan.id}/subdomains/`,
    },
    {
      key: "ips",
      label: t("runtimeDrawer.assets.liveHosts"),
      value: stats?.ipsCount ?? 0,
      icon: semanticIcons.concept.ip,
      href: `/scan/history/${scan.id}/ip-addresses/`,
    },
    {
      key: "websites",
      label: t("cards.websites"),
      value: stats?.websitesCount ?? 0,
      icon: semanticIcons.concept.website,
      href: `/scan/history/${scan.id}/websites/`,
    },
    {
      key: "pages",
      label: t("runtimeDrawer.summary.pages"),
      value: stats?.endpointsCount ?? 0,
      icon: semanticIcons.concept.endpoint,
      href: `/scan/history/${scan.id}/endpoints/`,
    },
    {
      key: "directories",
      label: t("cards.directories"),
      value: stats?.directoriesCount ?? 0,
      icon: semanticIcons.concept.directory,
      href: `/scan/history/${scan.id}/directories/`,
    },
    {
      key: "screenshots",
      label: t("cards.screenshots"),
      value: stats?.screenshotsCount ?? 0,
      icon: semanticIcons.concept.screenshot,
      href: `/scan/history/${scan.id}/screenshots/`,
    },
    {
      key: "risks",
      label: t("runtimeDrawer.summary.risks"),
      value: stats?.vulnsTotal ?? 0,
      icon: VulnerabilitiesMetricIcon,
      href: `/scan/history/${scan.id}/vulnerabilities/`,
    },
  ]

  return (
    <section className="space-y-3 border-t border-border/60 p-4">
      <h3 className={textRole.sectionTitle}>{t("runtimeDrawer.assets.title")}</h3>
      <div className="space-y-1">
        {items.map((item) => (
          <Link
            key={item.key}
            href={item.href}
            className="group flex min-w-0 items-center justify-between gap-3 rounded-md px-2 py-2 transition-colors hover:bg-accent/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <span className="flex min-w-0 items-center gap-2">
              <item.icon className="h-4 w-4 shrink-0 text-muted-foreground transition-colors group-hover:text-primary" />
              <span className={cn(textRole.metadataLabel, "truncate")}>{item.label}</span>
            </span>
            <span className={cn(textRole.metadataValueStrong, "shrink-0 tabular-nums")}>
              {item.value.toLocaleString()}
            </span>
          </Link>
        ))}
      </div>
    </section>
  )
}

function RuntimeSidePanelRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className={textRole.metadataLabel}>{label}</span>
      <span className={cn(textRole.metadataValueStrong, "min-w-0 text-right")}>{value}</span>
    </div>
  )
}

export function RuntimeOverviewSidePanel({
  scan,
  executedEngineNames,
  locale,
  t,
  statusLabel,
}: {
  scan: ScanRecord
  executedEngineNames: string[]
  locale: string
  t: TFn
  statusLabel: StatusLabelFn
}) {
  const status = scan.status ?? "pending"

  return (
    <aside className={SCAN_OVERVIEW_STICKY_SIDE_PANEL_CLASS}>
      <div className="overflow-hidden rounded-lg border border-border/60 bg-card">
        <section className="space-y-3 p-4">
          <h3 className={textRole.sectionTitle}>{t("runtimeDrawer.sidePanel.details")}</h3>
          <RuntimeSidePanelRow label={t("runtimeDrawer.meta.scanId")} value={`#${scan.id}`} />
          <RuntimeSidePanelRow
            label={t("runtimeDrawer.sidePanel.status")}
            value={
              <Badge
                variant={getScanStatusBadgeVariant(status)}
                className={cn("gap-1.5 px-2 py-0.5 text-xs", getScanStatusClasses(status))}
              >
                {statusIcon(status)}
                {statusLabel(status)}
              </Badge>
            }
          />
          <RuntimeSidePanelRow label={t("runtimeDrawer.meta.createdAt")} value={formatDateTime(scan.createdAt, locale)} />
        </section>

        <RuntimeAssetSummaryPanel scan={scan} t={t} />

        <section className="space-y-3 border-t border-border/60 p-4">
          <h3 className={textRole.sectionTitle}>{t("runtimeDrawer.sidePanel.modules")}</h3>
          <div className="space-y-2">
            {executedEngineNames.map((engineName) => (
              <div key={engineName} className="flex items-center justify-between gap-3">
                <span className={cn(textRole.metadataValue, "min-w-0 truncate")}>{engineName}</span>
                <semanticIcons.status.success className={cn("h-4 w-4 shrink-0", getScanStatusTextClass("succeeded"))} />
              </div>
            ))}
          </div>
        </section>
      </div>
    </aside>
  )
}

export function RuntimeTaskProgressLines({
  progressLogs,
  t,
}: {
  progressLogs: ScanLog[]
  t: TFn
}) {
  return (
    <div className="py-0.5">
      {progressLogs.length > 0 ? (
        <RawLogViewer
          content={buildScanLogContent(progressLogs)}
          className="h-auto max-h-48"
        />
      ) : (
        <p className={textRole.bodySubtle}>{t("runtimeDrawer.tasks.noProgress")}</p>
      )}
    </div>
  )
}

function RuntimeTaskFailureLines({ task, t }: { task: RuntimeTaskItem; t: TFn }) {
  const failureMessage = [task.failureSummary, task.failureDetail].filter(Boolean).join(" ")

  if (!failureMessage) {
    return null
  }

  return (
    <div className="space-y-2">
      <span className={cn(textRole.helperText, "tracking-normal")}>
        {t("runtimeDrawer.tasks.failureTitle")}
      </span>
      <p className={cn(textRole.bodySubtle, "break-words")}>{failureMessage}</p>
    </div>
  )
}

const diagnosticWatermarkFields: ReadonlyArray<{
  key: Exclude<keyof ResultTypeWatermark, "resultType">
  labelKey: string
}> = [
  { key: "receivedItems", labelKey: "receivedItems" },
  { key: "encodedItems", labelKey: "encodedItems" },
  { key: "submittedItems", labelKey: "submittedItems" },
  { key: "acknowledgedItems", labelKey: "acknowledgedItems" },
  { key: "submittedBatches", labelKey: "submittedBatches" },
  { key: "acknowledgedBatches", labelKey: "acknowledgedBatches" },
]

function getDiagnosticFailedStageLabel(stage: EngineDiagnosticFailedStage, t: TFn) {
  return t(`runtimeDrawer.tasks.diagnostics.failedStages.${stage}`)
}

function getDiagnosticErrorTypeLabel(errorType: EngineDiagnosticErrorType, t: TFn) {
  return t(`runtimeDrawer.tasks.diagnostics.errorTypes.${errorType}`)
}

function RuntimeTaskDiagnostics({ diagnostics, t }: { diagnostics?: EngineExecutionDiagnostics; t: TFn }) {
  if (!diagnostics) {
    return null
  }

  const hasWatermarks = Boolean(diagnostics.resultTypeWatermarks?.length)

  return (
    <section
      aria-label={t("runtimeDrawer.tasks.diagnostics.title")}
      className="space-y-3 border-t border-border/60 pt-3"
      data-runtime-task-diagnostics=""
    >
      <h4 className={textRole.sectionTitle}>{t("runtimeDrawer.tasks.diagnostics.title")}</h4>
      <dl className="grid grid-cols-1 gap-x-5 gap-y-2 sm:grid-cols-2">
        <div className="flex min-w-0 items-baseline justify-between gap-3">
          <dt className={textRole.helperText}>{t("runtimeDrawer.tasks.diagnostics.availability")}</dt>
          <dd className={cn(textRole.metadataValueStrong, "shrink-0")}>
            {diagnostics.availability === "available"
              ? t("runtimeDrawer.tasks.diagnostics.available")
              : t("runtimeDrawer.tasks.diagnostics.unavailable")}
          </dd>
        </div>
        <div className="flex min-w-0 items-baseline justify-between gap-3">
          <dt className={textRole.helperText}>{t("runtimeDrawer.tasks.diagnostics.resultState")}</dt>
          <dd className={cn(textRole.metadataValueStrong, "shrink-0")}>
            {t(`runtimeDrawer.tasks.diagnostics.resultStates.${diagnostics.resultState}`)}
          </dd>
        </div>
        {diagnostics.failedStage && diagnostics.errorType ? (
          <>
            <div className="flex min-w-0 items-baseline justify-between gap-3">
              <dt className={textRole.helperText}>{t("runtimeDrawer.tasks.diagnostics.failedStage")}</dt>
              <dd className={cn(textRole.metadataValueStrong, "min-w-0 break-words text-right")}>
                {getDiagnosticFailedStageLabel(diagnostics.failedStage, t)}
              </dd>
            </div>
            <div className="flex min-w-0 items-baseline justify-between gap-3">
              <dt className={textRole.helperText}>{t("runtimeDrawer.tasks.diagnostics.errorType")}</dt>
              <dd className={cn(textRole.metadataValueStrong, "min-w-0 break-words text-right")}>
                {getDiagnosticErrorTypeLabel(diagnostics.errorType, t)}
              </dd>
            </div>
          </>
        ) : null}
      </dl>

      {diagnostics.availability === "unavailable" ? (
        <p className={textRole.bodySubtle}>{t("runtimeDrawer.tasks.diagnostics.unavailableDescription")}</p>
      ) : hasWatermarks ? (
        <div className="space-y-3">
          <p className={textRole.helperText}>{t("runtimeDrawer.tasks.diagnostics.watermarks")}</p>
          {diagnostics.resultTypeWatermarks?.map((watermark) => (
            <div key={watermark.resultType} className="space-y-2 border-t border-border/60 pt-3 first:border-t-0 first:pt-0">
              <p className={cn(textRole.metadataValueStrong, "break-all")}>{watermark.resultType}</p>
              <dl className="grid grid-cols-2 gap-x-4 gap-y-2 sm:grid-cols-3">
                {diagnosticWatermarkFields.map((field) => (
                  <div key={field.key} className="flex min-w-0 items-baseline justify-between gap-2">
                    <dt className={textRole.helperText}>{t(`runtimeDrawer.tasks.diagnostics.${field.labelKey}`)}</dt>
                    <dd className={cn(textRole.metadataValueStrong, "shrink-0 tabular-nums")}>
                      {watermark[field.key].toLocaleString()}
                    </dd>
                  </div>
                ))}
              </dl>
            </div>
          ))}
        </div>
      ) : (
        <p className={textRole.bodySubtle}>
          {diagnostics.resultState === "complete"
            ? t("runtimeDrawer.tasks.diagnostics.zeroResultComplete")
            : t("runtimeDrawer.tasks.diagnostics.noWatermarks")}
        </p>
      )}
    </section>
  )
}

export function RuntimeTaskList({
  tasks,
  taskProgressLogsByTaskId,
  now,
  t,
  statusLabel,
}: {
  tasks: RuntimeTaskItem[]
  taskProgressLogsByTaskId: Map<number, ScanLog[]>
  now: Date
  t: TFn
  statusLabel: StatusLabelFn
}) {
  if (tasks.length === 0) {
    return (
      <div className={cn("rounded-lg border border-dashed border-border/60 px-4 py-6", textRole.bodySubtle)}>
        {t("runtimeDrawer.tasks.empty")}
      </div>
    )
  }

  const firstFailedTaskIndex = tasks.findIndex((task) => task.status === "failed")

  return (
    <section className="min-w-0 space-y-3">
      <h3 className={textRole.sectionTitle}>{t("runtimeDrawer.tasks.title")}</h3>
      <div className="min-w-0 space-y-2">
        {tasks.map((task, index) => {
          const hasNext = index < tasks.length - 1
          const metadataLabelClassName = textRole.helperText
          const progressLogs = taskProgressLogsByTaskId.get(task.id) ?? []
          const duration = getRuntimeTaskDurationSeconds(task, now)
          const presentationStatus = getRuntimeTaskPresentationStatus(task)
          const timelineNodeSurfaceClassName = presentationStatus === "not_started"
            ? "border border-border bg-card"
            : cn("border", getScanStatusSurfaceClass(task.status))

          return (
            <Collapsible
              key={task.id}
              defaultOpen={index === firstFailedTaskIndex}
              className="group/runtime-task relative min-w-0 pl-8 sm:pl-10"
            >
              {hasNext ? <div className="absolute bottom-[-16px] left-3 top-10 w-px bg-border" /> : null}

              <div
                className={cn(
                  "absolute left-0 top-3.5 z-10 flex h-6 w-6 items-center justify-center rounded-full shadow-sm",
                  timelineNodeSurfaceClassName
                )}
              >
                {statusIcon(presentationStatus)}
              </div>

              <div className="min-w-0 overflow-hidden rounded-md border border-border/60 bg-card transition-colors group-hover/runtime-task:border-primary/40 has-[button:focus-visible]:border-primary/40 has-[button:focus-visible]:ring-1 has-[button:focus-visible]:ring-primary/20">
                <CollapsibleTrigger
                  className="w-full text-left data-[panel-open]:[&_[data-runtime-task-chevron]]:rotate-90"
                  aria-label={t("runtimeDrawer.tasks.expand")}
                >
                  <div className="flex min-w-0 flex-col gap-3 px-4 py-3 lg:flex-row lg:items-center lg:justify-between">
                    <div className="min-w-0 flex-1 space-y-1.5">
                      <div className="flex flex-wrap items-center gap-2">
                        <p className={textRole.sectionTitle}>{task.title}</p>
                        <Badge
                          variant={getScanStatusBadgeVariant(task.status)}
                          className={cn("h-5 px-1.5 text-xs", getScanStatusClasses(task.status))}
                        >
                          {getRuntimeTaskStatusLabel(presentationStatus, t, statusLabel)}
                        </Badge>
                        {getRuntimeTaskSkipReasonLabel(task, t) ? (
                          <span className={textRole.helperText}>{getRuntimeTaskSkipReasonLabel(task, t)}</span>
                        ) : null}
                        {task.failureKind ? (
                          <Badge
                            variant={getScanStatusBadgeVariant("failed")}
                            className={cn("h-5 px-1.5 text-xs", getScanStatusClasses("failed"))}
                          >
                            {task.failureKind}
                          </Badge>
                        ) : null}
                      </div>
                      <p className={cn("max-w-2xl break-words", textRole.bodySubtle)}>
                        {task.detail}
                      </p>
                    </div>

                    <div className="flex shrink-0 items-center gap-3 lg:justify-end">
                      <div className="space-y-1.5 lg:w-56">
                        <div className="grid grid-cols-[3.5rem_minmax(0,1fr)] items-baseline gap-3">
                          <p className={cn(metadataLabelClassName, "text-right")}>{t("runtimeDrawer.tasks.duration")}</p>
                          <p className={cn(textRole.metadataValue, "text-left tabular-nums whitespace-nowrap")}>
                            {formatDurationClock(duration)}
                          </p>
                        </div>
                        <div className="grid grid-cols-[3.5rem_minmax(0,1fr)] items-baseline gap-3">
                          <p className={cn(metadataLabelClassName, "text-right")}>{t("runtimeDrawer.tasks.startedAt")}</p>
                          <p className={cn(textRole.metadataValue, "text-left tabular-nums whitespace-nowrap")}>
                            {formatTaskStartTime(task.startedAt)}
                          </p>
                        </div>
                      </div>
                      <ChevronRight
                        data-runtime-task-chevron=""
                        className="mt-0.5 h-4 w-4 text-muted-foreground transition-transform duration-200 ease-[cubic-bezier(0.22,1,0.36,1)] motion-reduce:transition-none"
                      />
                    </div>
                  </div>
                </CollapsibleTrigger>

                <CollapsibleContent className="border-t border-border/60 px-4 py-3">
                  <div className="space-y-3">
                    <RuntimeTaskDiagnostics diagnostics={task.diagnostics} t={t} />
                    <RuntimeTaskProgressLines progressLogs={progressLogs} t={t} />
                    <RuntimeTaskFailureLines task={task} t={t} />
                  </div>
                </CollapsibleContent>
              </div>
            </Collapsible>
          )
        })}
      </div>
    </section>
  )
}

export function RuntimeLogsPanel({
  logs,
  logsLoading,
  hasRuntimeLogs,
  skipReason,
  t,
}: {
  logs: ScanLog[]
  logsLoading: boolean
  hasRuntimeLogs: boolean
  skipReason?: string
  t: RuntimePanelTranslationFn
}) {
  if (!hasRuntimeLogs) {
    const reason = skipReason === "user_disabled"
      ? t("runtimeDrawer.details.noRuntimeLogsUserDisabled")
      : skipReason === "target_not_applicable"
        ? t("runtimeDrawer.details.noRuntimeLogsTargetNotApplicable")
        : t("runtimeDrawer.details.noRuntimeLogsPlanning")
    return <RuntimePanelFallback>{reason}</RuntimePanelFallback>
  }
  return (
    <div className={cn(RUNTIME_DETAIL_PANEL_CLASS, "bg-card")}>
      <ScanLogList logs={logs} loading={logsLoading} />
    </div>
  )
}

export function RuntimeConfigurationPanel({
  yamlContent,
  t,
}: {
  yamlContent: string
  t: RuntimePanelTranslationFn
}) {
  const tToast = useTranslations("toast")

  if (!yamlContent) {
    return <RuntimePanelFallback>{t("noConfig")}</RuntimePanelFallback>
  }

  return (
    <div className={cn(RUNTIME_DETAIL_PANEL_CLASS, "relative flex min-w-0 max-w-full flex-col bg-muted/20")}>
      <CopyButton
        value={yamlContent}
        copyLabel={t("runtimeDrawer.details.copyConfig")}
        copiedLabel={t("copied")}
        copyFailedLabel={tToast("copyFailed")}
        toastId="scan-runtime-configuration-copy"
        className="absolute right-2 top-2 z-10"
      />
      <div className="min-h-0 flex-1 overflow-auto">
        <pre className={cn(textRole.code, "min-h-full min-w-0 p-4 pr-12 leading-6 text-foreground/90")}>
          <code>{yamlContent}</code>
        </pre>
      </div>
    </div>
  )
}

export function RuntimePanelFallback({ children }: { children: React.ReactNode }) {
  return (
    <div className={cn(RUNTIME_DETAIL_PANEL_FALLBACK_CLASS, "flex items-center justify-center", textRole.bodySubtle)}>
      {children}
    </div>
  )
}

export function RuntimeDetailTabsPanel({
  t,
  detailTab,
  setDetailTab,
  logs,
  logsLoading,
  hasRuntimeLogs,
  skipReason,
  yamlContent,
  headerTrailing,
}: {
  t: RuntimePanelTranslationFn
  detailTab: RuntimeDetailTab
  setDetailTab: (value: RuntimeDetailTab) => void
  logs: ScanLog[]
  logsLoading: boolean
  hasRuntimeLogs: boolean
  skipReason?: string
  yamlContent: string
  headerTrailing?: React.ReactNode
}) {
  return (
    <section aria-label={t("runtimeDrawer.details.title")} className="min-w-0 overflow-hidden rounded-lg border border-border/60 bg-card">
      <DetailDrawerTabs value={detailTab} onValueChange={(value) => setDetailTab(value as RuntimeDetailTab)}>
        <div className="flex flex-col gap-2 border-b border-border px-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex min-w-0 flex-col gap-2 sm:flex-row sm:items-center">
            <DetailDrawerTabsList variant="minimal" size="md" className="min-w-0 max-w-full overflow-x-auto">
              <DetailDrawerTabsTrigger variant="minimal" activeIndicator="fixed" value="logs" size="md">
                {t("runtimeDrawer.details.allProgress")}
              </DetailDrawerTabsTrigger>
              <DetailDrawerTabsTrigger variant="minimal" activeIndicator="fixed" value="config" size="md">
                {t("runtimeDrawer.details.config")}
              </DetailDrawerTabsTrigger>
            </DetailDrawerTabsList>
          </div>

          {headerTrailing}
        </div>

        <DetailDrawerTabsContent value="logs">
          <RuntimeLogsPanel logs={logs} logsLoading={logsLoading} hasRuntimeLogs={hasRuntimeLogs} skipReason={skipReason} t={t} />
        </DetailDrawerTabsContent>

        <DetailDrawerTabsContent value="config">
          <RuntimeConfigurationPanel yamlContent={yamlContent} t={t} />
        </DetailDrawerTabsContent>
      </DetailDrawerTabs>
    </section>
  )
}

function ScanRuntimeDetailDrawerLoadingState() {
  return (
    <div {...getLoadingOwnerAttributes({ owner: "scan-runtime-detail-drawer", layer: "interaction", intent: "interaction" })} className="space-y-6">
      <section className="space-y-3">
        <div className="flex flex-wrap items-center gap-3">
          <Skeleton className="h-8 w-56" />
          <Skeleton className="h-5 w-16 rounded-full" />
          <Skeleton className="h-5 w-12 rounded-full" />
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <Skeleton className="h-4 w-32" />
          <Skeleton className="h-5 w-28 rounded-full" />
          <Skeleton className="h-5 w-24 rounded-full" />
          <Skeleton className="h-5 w-32 rounded-full" />
        </div>
        <div className="rounded-lg border border-border/60 bg-card/70 px-4 py-3">
          <div className="flex items-center gap-2">
            <Skeleton className="h-4 w-4 rounded-full" />
            <Skeleton className="h-4 w-28" />
          </div>
          <Skeleton className="mt-2 h-4 w-full" />
          <Skeleton className="mt-2 h-4 w-3/4" />
        </div>
      </section>

      <section className="space-y-3">
        <Skeleton className="h-5 w-24" />
        <div className="overflow-hidden rounded-lg border border-border/60 bg-card/70 px-3 py-2.5">
          <div className="flex items-center gap-3">
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-3 w-px" />
            <Skeleton className="h-4 w-20" />
            <Skeleton className="h-3 w-px" />
            <Skeleton className="h-4 w-20" />
            <Skeleton className="h-3 w-px" />
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-3 w-px" />
            <Skeleton className="h-4 w-20" />
            <Skeleton className="h-3 w-px" />
            <Skeleton className="h-4 w-20" />
          </div>
        </div>
        <div className="rounded-xl border border-border/60 bg-card/70">
          <div className="grid gap-0 md:grid-cols-3">
            <div className="flex items-center gap-3 px-4 py-3">
              <Skeleton className="h-4 w-20" />
              <Skeleton className="h-4 w-10" />
            </div>
            <div className="flex items-center gap-3 border-t border-border/60 px-4 py-3 md:border-l md:border-t-0">
              <Skeleton className="h-4 w-16" />
              <Skeleton className="h-4 w-10" />
            </div>
            <div className="flex items-center gap-3 border-t border-border/60 px-4 py-3 md:border-l md:border-t-0">
              <Skeleton className="h-4 w-16" />
              <Skeleton className="h-4 w-10" />
            </div>
          </div>
        </div>
      </section>

      <section className="space-y-3">
        <Skeleton className="h-5 w-24" />
        <div className="space-y-2.5">
          {Array.from({ length: 4 }).map((_, index) => (
            <div key={index} className="relative pl-9">
              <div className="absolute left-0 top-4 h-5 w-5 rounded-full border border-border/70 bg-background" />
              <div className="overflow-hidden rounded-md border border-border/60 bg-background px-4 py-3">
                <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                  <div className="min-w-0 space-y-2">
                    <div className="flex flex-wrap items-center gap-2">
                      <Skeleton className="h-5 w-32" />
                      <Skeleton className="h-5 w-10 rounded-full" />
                    </div>
                    <Skeleton className="h-4 w-full" />
                    <Skeleton className="h-4 w-3/4" />
                  </div>
                  <div className="flex shrink-0 items-start gap-4">
                    <div className="space-y-1">
                      <Skeleton className="h-3 w-16" />
                      <Skeleton className="h-4 w-20" />
                    </div>
                    <div className="space-y-1">
                      <Skeleton className="h-3 w-16" />
                      <Skeleton className="h-4 w-20" />
                    </div>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="space-y-3">
        <Skeleton className="h-5 w-28" />
        <div className="flex flex-col gap-2 border-b border-border px-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex min-w-0 flex-col gap-2 sm:flex-row sm:items-center">
            <DetailDrawerTabs value="logs" aria-hidden="true">
              <DetailDrawerTabsList variant="minimal" size="md" className="min-w-0 max-w-full overflow-x-auto">
                <DetailDrawerTabsTrigger variant="minimal" activeIndicator="fixed" value="logs" size="md" disabled>
                  <Skeleton className="h-3.5 w-20 rounded-full" />
                </DetailDrawerTabsTrigger>
                <DetailDrawerTabsTrigger variant="minimal" activeIndicator="fixed" value="config" size="md" disabled>
                  <Skeleton className="h-3.5 w-16 rounded-full" />
                </DetailDrawerTabsTrigger>
              </DetailDrawerTabsList>
            </DetailDrawerTabs>
          </div>
        </div>
        <div className="rounded-lg border border-border/60">
          <div className="space-y-2 px-4 py-4">
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-5/6" />
            <Skeleton className="h-4 w-2/3" />
            <Skeleton className="h-4 w-3/4" />
          </div>
        </div>
      </section>
    </div>
  )
}

export function ScanRuntimeDetailDrawer({ open, onOpenChange, scan }: ScanRuntimeDetailDrawerProps) {
  const t = useTranslations("scan.history.overview")
  const tStatus = useTranslations("common.status")
  const locale = useLocale()
  const [detailTab, setDetailTab] = React.useState<RuntimeDetailTab>("logs")
  const {
    scan: freshScan,
    isLoading,
    error,
    logs,
    logsLoading,
  } = useScanOverviewState({
    scanId: scan?.id || 0,
    t,
    logsEnabled: open,
    refreshEnabled: open,
  })
  const runtimeScan = freshScan || scan
  const executedEngineDisplay = useScanExecutedEngineDisplay(runtimeScan ? [runtimeScan] : [])
  const isInitialLoading = (isLoading && !runtimeScan) || (Boolean(runtimeScan) && executedEngineDisplay.isLoading)
  const runtimeTasks = React.useMemo(
    () => runtimeScan
      ? toRuntimeTaskItems(runtimeScan.runtimeTasks ?? [], runtimeScan, executedEngineDisplay.engineNamesById, executedEngineDisplay.engineDescriptionsById)
      : [],
    [executedEngineDisplay.engineDescriptionsById, executedEngineDisplay.engineNamesById, runtimeScan]
  )
  const runtimeDurationNow = useRuntimeDurationNow(hasLiveRuntimeDuration(runtimeScan, runtimeTasks))
  const taskProgressLogsByTaskId = React.useMemo(() => buildTaskProgressLogsByTaskId(logs), [logs])
  const hasRuntimeLogs = React.useMemo(
    () => Boolean(runtimeScan?.runtimeTasks?.some((task) => task.status !== "skipped" || task.startedAt || task.completedAt)),
    [runtimeScan?.runtimeTasks]
  )
  const planningSkipReason = React.useMemo(
    () => runtimeScan?.runtimeTasks?.find((task) => task.status === "skipped")?.skipReason,
    [runtimeScan?.runtimeTasks]
  )
  const yamlContent = React.useMemo(
    () => serializeWorkflowConfiguration(runtimeScan?.configuration),
    [runtimeScan?.configuration]
  )

  React.useEffect(() => {
    if (!open) {
      setDetailTab("logs")
    }
  }, [open])

  const runtimeContent = runtimeScan ? (
    <div className="min-w-0 max-w-full space-y-6">
      <RuntimeHeader scan={runtimeScan} tasks={runtimeTasks} now={runtimeDurationNow} locale={locale} t={t} statusLabel={tStatus} />
      <RuntimeSummarySection scan={runtimeScan} tasks={runtimeTasks} t={t} />
      <RuntimeTaskList tasks={runtimeTasks} taskProgressLogsByTaskId={taskProgressLogsByTaskId} now={runtimeDurationNow} t={t} statusLabel={tStatus} />

      <RuntimeDetailTabsPanel
        t={t}
        detailTab={detailTab}
        setDetailTab={setDetailTab}
        logs={logs}
        logsLoading={logsLoading}
        hasRuntimeLogs={hasRuntimeLogs}
        skipReason={planningSkipReason}
        yamlContent={yamlContent}
      />
    </div>
  ) : null

  const showErrorState = !isInitialLoading && (Boolean(error) || Boolean(executedEngineDisplay.error) || !runtimeScan)

  return (
    <DetailDrawer
      open={open}
      onOpenChange={onOpenChange}
      title={t("runtimeDrawer.title")}
    >
      <div className="min-w-0 flex-1 overflow-x-hidden overflow-y-auto px-6 py-5 [scrollbar-gutter:stable]">
        {showErrorState ? (
          <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border/60 py-16 text-center">
            <AlertTriangle className="mb-3 h-8 w-8 text-destructive" />
            <p className={textRole.bodySubtle}>{t("runtimeDrawer.loadError")}</p>
          </div>
        ) : (
          <ContentHandoff
            owner="scan-runtime-detail-drawer-content"
            isLoading={isInitialLoading}
            skeleton={<ScanRuntimeDetailDrawerLoadingState />}
          >
            {runtimeContent}
          </ContentHandoff>
        )}
      </div>
    </DetailDrawer>
  )
}
