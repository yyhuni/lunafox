"use client"

import { useMemo } from "react"
import { useTranslations } from "next-intl"

import { formatPageRefreshTimestamp } from "@/components/common/page-refresh-status-button"
import { semanticIcons } from "@/components/icons"
import {
  OVERVIEW_AGENT_LOCATION_MAP_BODY_CLASS,
  OVERVIEW_AGENT_LOCATION_MAP_HEADER_CLASS,
  OVERVIEW_AGENT_LOCATION_MAP_STATUS_LEGEND_CLASS,
  OVERVIEW_AGENT_LOCATION_MAP_SURFACE_CLASS,
  OverviewSectionPanel,
} from "@/components/overview/overview-section-layouts"
import { buildMapConnections } from "@/components/overview/agent-location-map-connections"
import { buildAgentLocationMapNodes } from "@/components/overview/agent-location-map-data"
import WorldMap, { type WorldMapMarker } from "@/components/overview/world-map"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { useAgentClusterSummary, useAgentLocationMap } from "@/hooks/use-agents"
import {
  getAgentStatusDistributionTone,
} from "@/lib/agent-status-distribution"
import { normalizeError } from "@/lib/errors/normalize-error"
import { getStatusToneColorVar, getStatusToneTextClass } from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

const MAP_LEGEND_STATUS_ORDER = ["healthy", "warning", "offline"] as const
const MAP_NETWORK_COLOR = "var(--muted-foreground)"
const MAP_REFRESH_INTERVAL_MS = 15000
const AgentIcon = semanticIcons.concept.agent
const WarningIcon = semanticIcons.status.warning

type AgentStatusLegendItem = {
  status: (typeof MAP_LEGEND_STATUS_ORDER)[number]
  count: number
  label: string
  color: string
}

function AgentStatusLegend({ items }: { items: readonly AgentStatusLegendItem[] }) {
  return (
    <div data-slot="agent-status-legend" className={OVERVIEW_AGENT_LOCATION_MAP_STATUS_LEGEND_CLASS}>
      {items.map((item) => (
        <span key={item.status} className="flex items-center gap-1.5">
          <span aria-hidden="true" className="size-2 rounded-full" style={{ backgroundColor: item.color }} />
          <span className={textRole.caption}>{item.label}</span>
          <span className={cn("tabular-nums", textRole.metadataValueStrong)}>{item.count}</span>
        </span>
      ))}
    </div>
  )
}

function AgentLocationMapLoading({
  label,
  statusItems,
}: {
  label: string
  statusItems?: readonly AgentStatusLegendItem[]
}) {
  return (
    <div
      aria-label={label}
      className={cn(OVERVIEW_AGENT_LOCATION_MAP_SURFACE_CLASS, "overflow-hidden aspect-[248/100] xl:aspect-auto")}
      role="status"
    >
      <Skeleton className="h-full w-full rounded-none opacity-35" />
      {statusItems?.length ? <AgentStatusLegend items={statusItems} /> : null}
    </div>
  )
}

function AgentLocationMapUnavailable({ message }: { message: string }) {
  return (
    <div className={cn(OVERVIEW_AGENT_LOCATION_MAP_SURFACE_CLASS, "flex items-center justify-center border border-dashed border-border/70 px-4 text-center aspect-[248/100] xl:aspect-auto")}>
      <p className={textRole.caption}>{message}</p>
    </div>
  )
}

function StaleSnapshotAlert({
  title,
  description,
  lastSuccessfulAt,
  retryLabel,
  onRetry,
}: {
  title: string
  description: (time: string) => string
  lastSuccessfulAt: number | null
  retryLabel: string
  onRetry: () => void | Promise<unknown>
}) {
  const time = lastSuccessfulAt
    ? formatPageRefreshTimestamp(new Date(lastSuccessfulAt))
    : "--"

  return (
    <Alert>
      <WarningIcon className={getStatusToneTextClass("warning")} />
      <AlertTitle>{title}</AlertTitle>
      <AlertDescription>
        <p>{description(time)}</p>
        <Button type="button" variant="outline" size="sm" onClick={() => void onRetry()}>
          {retryLabel}
        </Button>
      </AlertDescription>
    </Alert>
  )
}

function statusCount(
  status: (typeof MAP_LEGEND_STATUS_ORDER)[number],
  summary: NonNullable<ReturnType<typeof useAgentClusterSummary>["data"]>,
) {
  if (status === "healthy") return summary.healthyCount
  if (status === "warning") return summary.warningCount
  return summary.offlineCount
}

export function AgentLocationMap() {
  const t = useTranslations("overview.operational.agentLocationMap")
  const summary = useAgentClusterSummary({ refetchInterval: MAP_REFRESH_INTERVAL_MS })
  const locationMap = useAgentLocationMap({ refetchInterval: MAP_REFRESH_INTERVAL_MS })
  const nodes = useMemo(
    () => buildAgentLocationMapNodes(locationMap.data?.agents ?? []),
    [locationMap.data?.agents],
  )
  const serverLocation = locationMap.data?.serverLocation ?? null
  const connections = useMemo(
    () => serverLocation
      ? buildMapConnections({ lat: serverLocation.latitude, lng: serverLocation.longitude }, nodes)
      : [],
    [nodes, serverLocation],
  )
  const statusItems = summary.data
    ? MAP_LEGEND_STATUS_ORDER.map((status) => ({
        status,
        count: statusCount(status, summary.data),
        label: t(`status.${status}`),
        color: getStatusToneColorVar(getAgentStatusDistributionTone(status)),
      }))
    : []
  const markers = useMemo<WorldMapMarker[]>(
    () => [
      ...(serverLocation ? [{
        id: "server-location",
        lat: serverLocation.latitude,
        lng: serverLocation.longitude,
        color: MAP_NETWORK_COLOR,
        kind: "server" as const,
        locationState: serverLocation.state,
        label: t("serverLabel"),
        description: t("serverMarkerSummary", {
          ip: serverLocation.observedEgressIp,
        }),
        ariaLabel: t("serverAriaLabel", {
          freshness: t(`freshness.${serverLocation.state}`),
          ip: serverLocation.observedEgressIp,
          inference: t("inference"),
        }),
      }] : []),
      ...nodes.map((node) => ({
        id: node.id,
        lat: node.latitude,
        lng: node.longitude,
        color: node.color,
        kind: "agent" as const,
        locationState: node.locationState,
        label: node.name,
        description: t("agentMarkerSummary", {
          status: t(`status.${node.status}`),
          freshness: t(`freshness.${node.locationState}`),
        }),
        details: [
          t("observedSourceIp", { ip: node.sourceObservedIp }),
          t("inference"),
        ],
        ariaLabel: t("markerLabel", {
          name: node.name,
          status: t(`status.${node.status}`),
          freshness: t(`freshness.${node.locationState}`),
          ip: node.sourceObservedIp,
          inference: t("inference"),
        }),
      })),
    ],
    [nodes, serverLocation, t],
  )

  if (summary.isInitialError || (!summary.data && summary.isError)) {
    return (
      <OverviewSectionPanel className="h-full" contentClassName="xl:h-full">
        <AppErrorState
          error={normalizeError(summary.error, { notFoundKind: "unexpected-error" })}
          title={t("summaryLoadFailed")}
          description={t("summaryLoadFailedDescription")}
          onRetry={summary.refetch}
          variant="section"
        />
      </OverviewSectionPanel>
    )
  }

  return (
    <OverviewSectionPanel
      className="h-full"
      contentClassName="xl:h-full"
    >
      <div
        aria-busy={(summary.isPending && !summary.data) || (locationMap.isPending && !locationMap.data)}
        className="flex min-w-0 flex-col xl:h-full"
      >
        {summary.data ? (
          <div className={OVERVIEW_AGENT_LOCATION_MAP_HEADER_CLASS}>
            <div className="flex min-w-0 flex-wrap items-center gap-x-4 gap-y-1">
              <div className="flex items-center gap-2">
                <AgentIcon aria-hidden="true" className="size-4 shrink-0 text-muted-foreground" />
                <span className={textRole.caption}>{t("totalNodes")}</span>
                <span className={textRole.metricValueDisplay}>{summary.data.totalNodes}</span>
              </div>
              <div className="flex min-w-0 items-center gap-1.5">
                <span className={textRole.caption}>{t("executionCapacity")}</span>
                <span className={cn("shrink-0 tabular-nums", textRole.metadataValueStrong)}>
                  {summary.data.executionCapacity.availableSlots}/{summary.data.executionCapacity.configuredSlots}
                </span>
              </div>
              <div className="flex min-w-0 items-center gap-1.5">
                <span className={textRole.caption}>{t("coverageLabel")}</span>
                <span className={cn("shrink-0 tabular-nums", textRole.metadataValueStrong)}>
                  {summary.data.totalNodes > 0
                    ? t("coverageValue", {
                        positioned: summary.data.locationCoverage.positionedCount,
                        total: summary.data.totalNodes,
                      })
                    : t("coverageEmpty")}
                </span>
                {summary.data.totalNodes > 0 ? (
                  <span className={textRole.caption}>
                    {t("unpositioned", { count: summary.data.locationCoverage.unpositionedCount })}
                  </span>
                ) : null}
              </div>
              {summary.data.unknownCount > 0 ? (
                <span className={textRole.caption}>
                  {t("runtimeUnreported", { count: summary.data.unknownCount })}
                </span>
              ) : null}
            </div>
          </div>
        ) : null}

        {summary.isRefetchStale || locationMap.isRefetchStale ? (
          <div className="grid gap-2 pt-3">
            {summary.isRefetchStale ? (
              <StaleSnapshotAlert
                title={t("summaryStaleTitle")}
                description={(time) => t("summaryStaleDescription", { time })}
                lastSuccessfulAt={summary.lastSuccessfulAt}
                retryLabel={t("retrySummary")}
                onRetry={summary.refetch}
              />
            ) : null}
            {locationMap.isRefetchStale ? (
              <StaleSnapshotAlert
                title={t("mapStaleTitle")}
                description={(time) => t("mapStaleDescription", { time })}
                lastSuccessfulAt={locationMap.lastSuccessfulAt}
                retryLabel={t("retryMap")}
                onRetry={locationMap.refetch}
              />
            ) : null}
          </div>
        ) : null}

        <div className={OVERVIEW_AGENT_LOCATION_MAP_BODY_CLASS}>
          {locationMap.isPending && !locationMap.data ? (
            <AgentLocationMapLoading label={t("loading")} statusItems={statusItems} />
          ) : locationMap.isInitialError || (!locationMap.data && locationMap.isError) ? (
            <AppErrorState
              error={normalizeError(locationMap.error, { notFoundKind: "unexpected-error" })}
              title={t("loadFailed")}
              description={t("loadFailedDescription")}
              onRetry={locationMap.refetch}
              variant="section"
              className="min-h-48 py-4"
            />
          ) : markers.length > 0 ? (
            <div className={OVERVIEW_AGENT_LOCATION_MAP_SURFACE_CLASS}>
              <WorldMap
                ariaLabel={t("ariaLabel")}
                className="h-full min-h-0 aspect-[248/100] xl:aspect-auto"
                dots={connections}
                markers={markers}
                lineColor={MAP_NETWORK_COLOR}
              />
              {statusItems.length > 0 ? <AgentStatusLegend items={statusItems} /> : null}
            </div>
          ) : (
            <AgentLocationMapUnavailable message={t("unavailable")} />
          )}
        </div>
      </div>
    </OverviewSectionPanel>
  )
}
