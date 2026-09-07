"use client"

import { useMemo } from "react"
import { useNow, useTranslations } from "next-intl"
import {
  IconSettings,
  IconTrash,
  IconActivity,
  IconHeartbeat,
  IconClock,
  IconCpu,
  IconDatabase,
  IconTerminal,
  HardDrive,
  semanticIcons,
} from "@/components/icons"
import { Badge } from "@/components/ui/badge"
import { Card } from "@/components/ui/card"
import { Status, StatusLabel } from "@/components/shared/feedback/status"
import {
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu"
import { DenseRowActionMenu } from "@/components/shared/data-table/menu-owners"
import { useFormatNumber, useFormatHeartbeatTime } from "@/lib/i18n-format"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type { Agent } from "@/types/agent.types"
import { SegmentedMetricProgress } from "@/components/shared/metrics/segmented-metric-progress"
import {
  getAgentRuntimeShellClass,
  getAgentHeartbeatTextClass,
  getAgentRuntimeStatus,
} from "./agent-status"
import { getAgentConnectionIpDisplay } from "./agent-connection-ip"
import {
  AGENT_CARD_BODY_CLASS,
  AGENT_CARD_FOOTER_CLASS,
  AGENT_CARD_HEADER_CLASS,
  AGENT_CARD_HEADER_CONTENT_CLASS,
  AGENT_CARD_IP_CLASS,
  AGENT_CARD_INFO_GRID_CLASS,
  AGENT_CARD_METRICS_CLASS,
  AGENT_CARD_SHELL_CLASS,
  AGENT_CARD_TITLE_CLASS,
} from "./agent-layout-contract"
import { AgentCardCompactLoadingState } from "./agent-card-compact-loading-state"

const TaskSlotsIcon = semanticIcons.metric.taskSlots

function formatUptime(seconds?: number | null) {
  if (seconds === null || seconds === undefined) return "-"
  const total = Math.max(0, Math.floor(seconds))
  const minutes = Math.floor(total / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)
  if (days > 0) return `${days}d ${hours % 24}h`
  if (hours > 0) return `${hours}h ${minutes % 60}m`
  return `${minutes}m`
}

interface AgentCardCompactResolvedProps {
  agentNode: Agent
  onConfig: (agentNode: Agent) => void
  onDelete: (agentNode: Agent) => void
  onLogs: (agentNode: Agent) => void
  loading?: false
}

interface AgentCardCompactLoadingProps {
  loading: true
}

type AgentCardCompactProps = AgentCardCompactResolvedProps | AgentCardCompactLoadingProps

function normalizePastDate(value: string | Date | null | undefined, nowMs: number): Date | null {
  if (!value) return null
  const parsed = value instanceof Date ? value : new Date(value)
  const parsedMs = parsed.getTime()
  if (Number.isNaN(parsedMs)) return null
  return new Date(Math.min(parsedMs, nowMs))
}

function AgentCardCompactResolved({
  agentNode,
  onConfig,
  onDelete,
  onLogs,
}: AgentCardCompactResolvedProps) {
  const t = useTranslations("settings.agents")
  const formatHeartbeatTime = useFormatHeartbeatTime(7, "compact")
  const now = useNow({ updateInterval: 5000 })
  const formatNumber = useFormatNumber()

  const heartbeat = agentNode.heartbeat
  const nowMs = now.getTime()
  const lastSeenDate = useMemo(() => normalizePastDate(agentNode.lastHeartbeat, nowMs), [agentNode.lastHeartbeat, nowMs])

  const { display: lastSeenText, title: lastSeenTitle } = formatHeartbeatTime(lastSeenDate)
  const statusVariant = getAgentRuntimeStatus(agentNode.status)
  const statusLabel = statusVariant === "online"
    ? t("status.online")
    : statusVariant === "offline"
      ? t("status.offline")
      : statusVariant === "maintenance"
        ? t("status.maintenance")
        : t("status.unknown")
  const connectionIpDisplay = getAgentConnectionIpDisplay(agentNode, {
    offline: t("connectionIp.offline"),
    unobserved: t("connectionIp.unobserved"),
  })

  return (
    <Card className={cn(
      AGENT_CARD_SHELL_CLASS,
      "hover:shadow-md",
      getAgentRuntimeShellClass(agentNode.status)
    )}>
      <div className={AGENT_CARD_HEADER_CLASS}>
        <div className={AGENT_CARD_HEADER_CONTENT_CLASS}>
          <Status status={statusVariant} className="shrink-0 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide">
            <StatusLabel>{statusLabel}</StatusLabel>
          </Status>
          <div className="min-w-0">
            <div className={AGENT_CARD_TITLE_CLASS} title={agentNode.displayName}>
              {agentNode.displayName}
            </div>
            <div className={AGENT_CARD_IP_CLASS} title={connectionIpDisplay}>
              {connectionIpDisplay}
            </div>
          </div>
        </div>

        <DenseRowActionMenu
          ariaLabel={t("actions.title")}
          align="start"
          ownerClassName="shrink-0"
          triggerSize="icon-sm"
        >
          <DropdownMenuLabel>{t("actions.title")}</DropdownMenuLabel>
          <DropdownMenuItem onClick={() => onConfig(agentNode)}>
            <IconSettings className="h-4 w-4" />
            {t("actions.config")}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => onLogs(agentNode)}>
            <IconTerminal className="h-4 w-4" />
            {t("actions.logs")}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem variant="destructive" onClick={() => onDelete(agentNode)}>
            <IconTrash className="h-4 w-4" />
            {t("actions.delete")}
          </DropdownMenuItem>
        </DenseRowActionMenu>
      </div>

      <div className={AGENT_CARD_BODY_CLASS}>
        <div className={AGENT_CARD_INFO_GRID_CLASS}>
          <div className="min-w-0 space-y-0.5">
            <span className={textRole.helperText}>{t("card.host")}</span>
            <div className="font-mono text-xs truncate" title={agentNode.observedHostname || t("unknownHost")}>
              {agentNode.observedHostname || t("unknownHost")}
            </div>
          </div>

          <div className="min-w-0 space-y-0.5 text-right">
            <span className={textRole.helperText}>{t("card.version")}</span>
            <div className="font-mono text-xs truncate" title={agentNode.agentVersion || "-"}>
              {agentNode.agentVersion || "-"}
            </div>
          </div>
        </div>

        {heartbeat ? (
          <div className={AGENT_CARD_METRICS_CLASS}>
            <SegmentedMetricProgress
              label={t("metrics.cpu")}
              value={heartbeat.cpu}
              threshold={agentNode.cpuThreshold}
              icon={<IconCpu className="h-3 w-3" />}
            />
            <SegmentedMetricProgress
              label={t("metrics.mem")}
              value={heartbeat.mem}
              threshold={agentNode.memThreshold}
              icon={<IconDatabase className="h-3 w-3" />}
            />
            <SegmentedMetricProgress
              label={t("metrics.disk")}
              value={heartbeat.disk}
              threshold={agentNode.diskThreshold}
              icon={<HardDrive className="h-3 w-3" />}
            />
          </div>
        ) : (
          <div
            className={cn("rounded border border-border border-dashed bg-muted/20 py-6 text-center", textRole.helperText)}
            data-testid="agent-realtime-metrics-unavailable"
          >
            {t("card.realtimeMetricsUnavailable")}
          </div>
        )}
      </div>

      <div className={cn(AGENT_CARD_FOOTER_CLASS, textRole.helperText)}>
        {heartbeat ? (
          <>
            <div className="grid grid-cols-2 gap-3">
              <div className="flex min-w-0 items-center justify-between gap-2" title={t("metrics.runningTasks")}>
                <span className="flex min-w-0 items-center gap-1 text-muted-foreground">
                  <IconActivity className="h-2.5 w-2.5" />
                  {t("metrics.runningTasks")}
                </span>
                <Badge variant="secondary" className="bg-muted border-border font-mono h-5 px-1.5 text-[10px] text-foreground">
                  {formatNumber.formatInteger(heartbeat.runningTasks)}
                </Badge>
              </div>
              <div className="flex min-w-0 items-center justify-between gap-2" title={t("metrics.usedTaskSlots")}>
                <span className="flex min-w-0 items-center gap-1 text-muted-foreground">
                  <TaskSlotsIcon className="h-2.5 w-2.5" />
                  {t("metrics.usedTaskSlots")}
                </span>
                <Badge variant="secondary" className="bg-muted border-border font-mono h-5 px-1.5 text-[10px] text-foreground">
                  {formatNumber.formatInteger(heartbeat.taskSlotsUsed)}
                  <span className="mx-0.5 opacity-40">/</span>
                  {agentNode.maxTasks}
                </Badge>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="flex min-w-0 items-center justify-between gap-2" title={lastSeenTitle}>
                <span className="flex shrink-0 items-center gap-1 text-muted-foreground">
                  <IconHeartbeat className="h-2.5 w-2.5" />
                  {t("metrics.lastHeartbeat")}
                </span>
                <span className={cn("min-w-0 truncate text-right font-medium tabular-nums", getAgentHeartbeatTextClass())}>
                  {lastSeenText}
                </span>
              </div>
              <div className="flex min-w-0 items-center justify-between gap-2" title={t("card.uptime")}>
                <span className="flex shrink-0 items-center gap-1 text-muted-foreground">
                  <IconClock className="h-2.5 w-2.5" />
                  {t("card.uptime")}
                </span>
                <span className="font-medium tabular-nums text-foreground">{formatUptime(heartbeat.uptime)}</span>
              </div>
            </div>
          </>
        ) : (
          <div className="flex min-w-0 items-center justify-between gap-2" title={lastSeenTitle}>
            <span className="flex shrink-0 items-center gap-1 text-muted-foreground">
              <IconHeartbeat className="h-2.5 w-2.5" />
              {t("metrics.lastHeartbeat")}
            </span>
            <span className={cn("min-w-0 truncate text-right font-medium tabular-nums", getAgentHeartbeatTextClass())}>
              {lastSeenText}
            </span>
          </div>
        )}
      </div>
    </Card>
  )
}

export function AgentCardCompact(props: AgentCardCompactProps) {
  if (props.loading) {
    return <AgentCardCompactLoadingState />
  }

  return <AgentCardCompactResolved {...props} />
}
