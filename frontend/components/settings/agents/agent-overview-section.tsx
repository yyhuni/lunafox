"use client"

import type { ReactNode } from "react"
import { useTranslations } from "next-intl"

import { PageRefreshStatusButton, formatPageRefreshTimestamp } from "@/components/common/page-refresh-status-button"
import { semanticIcons } from "@/components/icons"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { getLoadingStructureSlotAttributes } from "@/components/shared/loading/loading-owner"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { normalizeError } from "@/lib/errors/normalize-error"
import {
  getStatusToneBgClass,
  getStatusToneSurfaceClass,
  getStatusToneTextClass,
  type StatusTone,
} from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type {
  AgentClusterExecutionCapacity,
  AgentClusterState,
  AgentClusterSummary,
} from "@/types/agent.types"
import {
  AGENT_CLUSTER_CAPACITY_SUMMARY_CLASS,
  AGENT_CLUSTER_METRICS_CLASS,
  AGENT_CLUSTER_SUMMARY_CELL_DIVIDER_CLASS,
  AGENT_CLUSTER_SUMMARY_ROOT_CLASS,
  AGENT_CLUSTER_SUMMARY_STATE_CLASS,
  AGENT_LIST_OVERVIEW_REGION_SLOT,
  AGENT_OVERVIEW_HEADER_CLASS,
  AGENT_OVERVIEW_TITLE_GROUP_CLASS,
} from "./agent-layout-contract"

type AgentOverviewSectionProps =
  | {
      loading: true
    }
  | {
      loading?: false
      summary: AgentClusterSummary | undefined
      error: unknown
      isInitialError: boolean
      isRefetchStale: boolean
      lastSuccessfulAt: number | null
      lastRefreshedAt: Date | null
      isRefreshing: boolean
      onRefresh: () => void | Promise<void>
      onRetry: () => void | Promise<unknown>
    }

type CapacityPart = {
  key: "occupiedSlots" | "availableSlots" | "unavailableSlots"
  value: number
  tone: StatusTone
}

const AgentIcon = semanticIcons.concept.agent
const WarningIcon = semanticIcons.status.warning

const LOADING_SUMMARY: AgentClusterSummary = {
  resourceName: "agentClusterSummaries/current",
  generatedAt: "2026-01-01T00:00:00Z",
  executionFreshnessSeconds: 15,
  totalNodes: 10,
  healthyCount: 6,
  warningCount: 2,
  offlineCount: 2,
  unknownCount: 0,
  staleAgentCount: 0,
  executionCapacity: {
    configuredSlots: 70,
    occupiedSlots: 31,
    availableSlots: 30,
    unavailableSlots: 9,
    overcommittedSlots: 0,
  },
  clusterState: "needsAttention",
  reasonCodes: ["offline_agents", "warning_agents"],
  locationCoverage: {
    positionedCount: 8,
    unpositionedCount: 2,
  },
}

function InlineLoadingText({ text }: { text: string }) {
  return (
    <span aria-hidden="true" className="relative inline-block max-w-full select-none text-transparent">
      {text}
      <span className="loading-skeleton !absolute inset-y-0 left-0 block w-full rounded-md" />
    </span>
  )
}

function getClusterStatePresentation(state: AgentClusterState) {
  if (state === "critical") {
    return { tone: "error" as const, Icon: semanticIcons.status.error }
  }

  if (state === "needsAttention") {
    return { tone: "warning" as const, Icon: semanticIcons.status.warning }
  }

  if (state === "healthy") {
    return { tone: "success" as const, Icon: semanticIcons.status.success }
  }

  return { tone: "muted" as const, Icon: semanticIcons.status.unknown }
}

function formatPercentage(value: number, total: number) {
  if (total <= 0) return "0%"

  return String(Math.round((value / total) * 1000) / 10) + "%"
}

function ClusterNodeSummary({
  summary,
  loading,
}: {
  summary: AgentClusterSummary
  loading: boolean
}) {
  const t = useTranslations("settings.agents")
  const metrics = [
    { key: "healthy", value: summary.healthyCount, tone: "success" as const },
    { key: "warning", value: summary.warningCount, tone: "warning" as const },
    { key: "offline", value: summary.offlineCount, tone: "error" as const },
    { key: "unknown", value: summary.unknownCount, tone: "muted" as const },
  ]

  return (
    <div className={AGENT_CLUSTER_SUMMARY_STATE_CLASS}>
      <div className="flex min-w-0 items-center gap-3">
        <span
          aria-hidden="true"
          className="radius-round flex size-10 shrink-0 items-center justify-center border border-border bg-muted/30 text-muted-foreground"
        >
          <AgentIcon className="size-5" />
        </span>

        <div className="min-w-0">
          <div className="flex flex-wrap items-baseline gap-x-2 gap-y-1">
            <span data-featured="true" className={textRole.metricValueDisplay}>
              {loading ? <InlineLoadingText text={String(summary.totalNodes)} /> : String(summary.totalNodes)}
            </span>
            <span className={textRole.bodyStrong}>{t("overview.agentNodes")}</span>
          </div>
          <div className={cn("mt-2", AGENT_CLUSTER_METRICS_CLASS, textRole.helperText)}>
            {metrics.map((metric) => (
              <span key={metric.key} className="whitespace-nowrap">
                <span className={cn("mr-1.5 inline-block size-2 rounded-full", getStatusToneBgClass(metric.tone))} />
                {loading
                  ? <InlineLoadingText text={t("stats." + metric.key) + " " + metric.value} />
                  : t("stats." + metric.key) + " " + metric.value}
              </span>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}

function ClusterAttentionSummary({
  summary,
  loading,
}: {
  summary: AgentClusterSummary
  loading: boolean
}) {
  const t = useTranslations("settings.agents")
  const presentation = getClusterStatePresentation(summary.clusterState)
  const StateIcon = presentation.Icon
  const detail = summary.reasonCodes.length > 0
    ? summary.reasonCodes.map((reasonCode) => t("overview.clusterState.reasons." + reasonCode)).join(" · ")
    : t("overview.clusterState.noAttention")

  return (
    <div className={AGENT_CLUSTER_SUMMARY_CELL_DIVIDER_CLASS}>
      <div className="flex min-w-0 items-start gap-2">
        <span
          aria-hidden="true"
          className={cn(
            "radius-round flex size-8 shrink-0 items-center justify-center",
            getStatusToneSurfaceClass(presentation.tone),
          )}
        >
          <StateIcon className={cn("size-4", getStatusToneTextClass(presentation.tone))} />
        </span>
        <div className="min-w-0">
          <div className="flex flex-wrap items-baseline gap-x-2 gap-y-1">
            <span className={cn(textRole.panelTitle, getStatusToneTextClass(presentation.tone))}>
              {loading
                ? <InlineLoadingText text={t("overview.clusterState.healthy")} />
                : t("overview.clusterState." + summary.clusterState)}
            </span>
          </div>
          <p className={cn("mt-1", textRole.helperText)}>
            {loading ? <InlineLoadingText text={detail} /> : detail}
          </p>
        </div>
      </div>
    </div>
  )
}

function HealthyNodeSummary({
  summary,
  loading,
}: {
  summary: AgentClusterSummary
  loading: boolean
}) {
  const t = useTranslations("settings.agents")
  const percentage = formatPercentage(summary.healthyCount, summary.totalNodes)
  const value = summary.totalNodes > 0
    ? String(summary.healthyCount) + "/" + String(summary.totalNodes)
    : "0"

  return (
    <div className={AGENT_CLUSTER_SUMMARY_CELL_DIVIDER_CLASS}>
      <div className="flex w-full items-baseline">
        <span className={textRole.sectionTitle}>{t("overview.healthyNodes")}</span>
      </div>

      <div className="mt-2 flex w-full items-baseline justify-between gap-3">
        <span className={textRole.metricValueDisplay}>
          {loading ? <InlineLoadingText text={value} /> : value}
        </span>
        <span className={cn("shrink-0", textRole.helperText)}>
          {loading ? <InlineLoadingText text={percentage} /> : percentage}
        </span>
      </div>
      <div
        aria-label={t("overview.healthyNodes")}
        aria-valuemax={summary.totalNodes}
        aria-valuemin={0}
        aria-valuenow={summary.healthyCount}
        className="radius-pill mt-1 flex h-2 w-full min-w-0 overflow-hidden bg-muted"
        role="progressbar"
      >
        <span
          className={cn("h-full", getStatusToneBgClass("success"))}
          style={{ width: (summary.totalNodes > 0 ? (summary.healthyCount / summary.totalNodes) * 100 : 0) + "%" }}
        />
      </div>
    </div>
  )
}

function CompactCapacitySummary({
  capacity,
  loading,
}: {
  capacity: AgentClusterExecutionCapacity
  loading: boolean
}) {
  const t = useTranslations("settings.agents")
  const parts: CapacityPart[] = [
    { key: "availableSlots", value: capacity.availableSlots, tone: "success" },
    { key: "occupiedSlots", value: capacity.occupiedSlots, tone: "info" },
    { key: "unavailableSlots", value: capacity.unavailableSlots, tone: "muted" },
  ]
  const visualTotal = Math.max(capacity.configuredSlots, 1)
  const availablePercentage = formatPercentage(capacity.availableSlots, capacity.configuredSlots)

  return (
    <div className={AGENT_CLUSTER_CAPACITY_SUMMARY_CLASS}>
      <div className="flex w-full items-baseline">
        <span className={textRole.sectionTitle}>{t("overview.executionCapacity")}</span>
      </div>

      <div className="mt-2 flex w-full min-w-0 items-baseline justify-between gap-3">
        <div className="flex min-w-0 items-baseline gap-2">
          <span className={textRole.metricValueDisplay}>
            {loading
              ? <InlineLoadingText text={String(capacity.availableSlots) + "/" + String(capacity.configuredSlots)} />
              : String(capacity.availableSlots) + "/" + String(capacity.configuredSlots)}
          </span>
        </div>
        <span className={cn("shrink-0", textRole.helperText)}>
          {loading ? <InlineLoadingText text={availablePercentage} /> : availablePercentage} {t("overview.availableSlots")}
        </span>
      </div>
      <div
        aria-label={t("overview.executionCapacity")}
        className="radius-pill mt-1 flex h-2 w-full min-w-0 overflow-hidden bg-muted"
        role="img"
      >
        {parts.map((part) => part.value > 0 ? (
          <span
            key={part.key}
            className={cn("h-full", getStatusToneBgClass(part.tone))}
            style={{ width: (part.value / visualTotal) * 100 + "%" }}
            title={t("overview." + part.key) + ": " + part.value}
          />
        ) : null)}
      </div>
      <div className={cn("mt-2 flex w-full flex-wrap gap-x-3 gap-y-1", textRole.helperText)}>
        {parts.map((part) => (
          <span key={part.key} className="whitespace-nowrap">
            <span className={cn("mr-1.5 inline-block size-2 rounded-full", getStatusToneBgClass(part.tone))} />
            {t("overview." + part.key)} {part.value}
          </span>
        ))}
        {capacity.overcommittedSlots > 0 ? (
          <span className={cn("whitespace-nowrap", getStatusToneTextClass("warning"))}>
            {t("overview.overcommittedSlots")} {capacity.overcommittedSlots}
          </span>
        ) : null}
      </div>
    </div>
  )
}

function AgentOverviewHeader({
  loading,
  refreshControl,
}: {
  loading: boolean
  refreshControl: ReactNode
}) {
  const t = useTranslations("settings.agents")
  const clusterMapLabel = t("overview.clusterMap")
  const resolvedClusterMapLabel = clusterMapLabel === "overview.clusterMap" ? "CLUSTER_MAP" : clusterMapLabel

  return (
    <div className={AGENT_OVERVIEW_HEADER_CLASS}>
      <div className={AGENT_OVERVIEW_TITLE_GROUP_CLASS}>
        <h2 className={textRole.pageTitle}>
          {loading ? <InlineLoadingText text={t("overview.title")} /> : t("overview.title")}
        </h2>
        <span className={cn(textRole.monoLabel, "shrink-0")}>/ {resolvedClusterMapLabel}</span>
      </div>
      {refreshControl}
    </div>
  )
}

function SummaryStaleAlert({
  lastSuccessfulAt,
  onRetry,
}: {
  lastSuccessfulAt: number | null
  onRetry: () => void | Promise<unknown>
}) {
  const t = useTranslations("settings.agents")
  const formattedTime = lastSuccessfulAt
    ? formatPageRefreshTimestamp(new Date(lastSuccessfulAt))
    : t("overview.updatedAtUnavailable")

  return (
    <Alert data-testid="agent-overview-stale-alert">
      <WarningIcon className={getStatusToneTextClass("warning")} />
      <AlertTitle>{t("overview.stale.title")}</AlertTitle>
      <AlertDescription>
        <p>{t("overview.stale.description", { time: formattedTime })}</p>
        <Button type="button" variant="outline" size="sm" onClick={() => void onRetry()}>
          {t("overview.stale.retry")}
        </Button>
      </AlertDescription>
    </Alert>
  )
}

function AgentOverviewLayout({
  loading,
  summary,
  refreshControl,
  staleNotice,
}: {
  loading: boolean
  summary: AgentClusterSummary
  refreshControl: ReactNode
  staleNotice?: ReactNode
}) {
  return (
    <div
      {...getLoadingStructureSlotAttributes(AGENT_LIST_OVERVIEW_REGION_SLOT)}
      data-testid={loading ? "agent-overview-loading-state" : "agent-overview-section"}
      className="space-y-4"
    >
      <AgentOverviewHeader loading={loading} refreshControl={refreshControl} />
      {staleNotice}
      <div className={AGENT_CLUSTER_SUMMARY_ROOT_CLASS}>
        <ClusterNodeSummary summary={summary} loading={loading} />
        <ClusterAttentionSummary summary={summary} loading={loading} />
        <HealthyNodeSummary summary={summary} loading={loading} />
        <CompactCapacitySummary capacity={summary.executionCapacity} loading={loading} />
      </div>
    </div>
  )
}

export function AgentOverviewSection(props: AgentOverviewSectionProps) {
  const t = useTranslations("settings.agents")

  if (props.loading) {
    return (
      <AgentOverviewLayout
        loading
        summary={LOADING_SUMMARY}
        refreshControl={<ActionSkeleton size="sm" widthClassName="w-40" />}
      />
    )
  }

  const refreshControl = (
    <PageRefreshStatusButton
      isRefreshing={props.isRefreshing}
      lastRefreshedAt={props.lastRefreshedAt}
      onRefresh={props.onRefresh}
      refreshLabel={t("overview.refresh")}
      updatedAtLabel={t("overview.updatedAtLabel")}
      updatedAtUnavailable={t("overview.updatedAtUnavailable")}
    />
  )

  if (props.isInitialError || !props.summary) {
    return (
      <div
        {...getLoadingStructureSlotAttributes(AGENT_LIST_OVERVIEW_REGION_SLOT)}
        data-testid="agent-overview-error-state"
        className="space-y-4"
      >
        <AgentOverviewHeader loading={false} refreshControl={refreshControl} />
        <AppErrorState
          error={normalizeError(props.error, { notFoundKind: "unexpected-error" })}
          title={t("overview.loadFailed")}
          description={t("overview.loadFailedDescription")}
          onRetry={props.onRetry}
          variant="section"
        />
      </div>
    )
  }

  return (
    <AgentOverviewLayout
      loading={false}
      summary={props.summary}
      refreshControl={refreshControl}
      staleNotice={props.isRefetchStale ? (
        <SummaryStaleAlert lastSuccessfulAt={props.lastSuccessfulAt} onRetry={props.onRetry} />
      ) : undefined}
    />
  )
}
