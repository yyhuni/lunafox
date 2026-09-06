"use client"

import { useMemo } from "react"
import { useTranslations } from "next-intl"
import {
  IconSettings,
  IconTrash,
  IconAlertTriangle,
  IconActivity,
} from "@/components/icons"
import { Badge } from "@/components/ui/badge"
import {
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu"
import { DenseRowActionMenu } from "@/components/shared/data-table/menu-owners"
import { Status, StatusIndicator } from "@/components/shared/feedback/status"
import { useFormatNumber, useFormatHeartbeatTime } from "@/lib/i18n-format"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type { Agent } from "@/types/agent.types"
import {
  getAgentHealthBadgeVariant,
  getAgentHealthClassName,
  getAgentMetricAlertTextClass,
  getAgentMetricBarClass,
  getAgentMetricStatus,
  getAgentMetricTextClass,
  getAgentRuntimeBorderClass,
  getAgentRuntimeShellClass,
  getAgentHeartbeatTextClass,
  getAgentRuntimeStatus,
  getAgentRuntimeTextClass,
} from "./agent-status"
import { getAgentConnectionIpDisplay } from "./agent-connection-ip"

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

interface MetricBarProps {
  label: string
  value: number
  threshold?: number
  className?: string
}

function MetricBar({ label, value, threshold, className }: MetricBarProps) {
  const percentage = Math.min(100, Math.max(0, value))
  const status = useMemo(() => getAgentMetricStatus(percentage, threshold), [percentage, threshold])
  const progressColor = useMemo(() => getAgentMetricBarClass(percentage, threshold), [percentage, threshold])
  const valueClassName = useMemo(() => getAgentMetricTextClass(percentage, threshold), [percentage, threshold])
  const alertTextClassName = useMemo(() => getAgentMetricAlertTextClass(percentage, threshold), [percentage, threshold])

  return (
    <div className={cn("flex items-center gap-1.5", className)}>
      <span className="min-w-[32px] text-[10px] text-muted-foreground">{label}</span>
      <div className="bg-muted h-1.5 overflow-hidden relative rounded-full w-16">
        <div
          className={cn("h-full w-full origin-left transition-[transform,background-color] duration-300", progressColor)}
          style={{ transform: `scaleX(${percentage / 100})` }}
        />
      </div>
      <span className={cn("text-[10px] font-medium tabular-nums text-right", valueClassName)}>
        {percentage.toFixed(0)}%
        {threshold && (
          <span className="text-muted-foreground">/{threshold}%</span>
        )}
      </span>
      {status !== "normal" && (
        <IconAlertTriangle className={cn("h-3 shrink-0 w-3", alertTextClassName)} />
      )}
    </div>
  )
}

interface AgentListItemProps {
  agentNode: Agent
  onConfig: (agentNode: Agent) => void
  onDelete: (agentNode: Agent) => void
}

export function AgentListItem({
  agentNode,
  onConfig,
  onDelete,
}: AgentListItemProps) {
  const t = useTranslations("settings.agents")
  const formatHeartbeatTime = useFormatHeartbeatTime()
  const formatNumber = useFormatNumber()
  const connectionIpDisplay = getAgentConnectionIpDisplay(agentNode, {
    offline: t("connectionIp.offline"),
    unobserved: t("connectionIp.unobserved"),
  })


  const healthState = (agentNode.health?.state || "unknown").toLowerCase()
  const healthLabel = useMemo(() => {
    if (healthState === "ok") return t("health.ok")
    if (healthState === "warning" || healthState === "warn") return t("health.warning")
    if (healthState === "error" || healthState === "critical") return t("health.warning")
    return t("health.unknown")
  }, [healthState, t])

  const heartbeat = agentNode.heartbeat

  // Check if any metric exceeds the threshold
  const hasWarnings = useMemo(() => {
    if (!heartbeat) return false
    return (
      heartbeat.cpu >= agentNode.cpuThreshold ||
      heartbeat.mem >= agentNode.memThreshold ||
      heartbeat.disk >= agentNode.diskThreshold
    )
  }, [heartbeat, agentNode])

  // Calculate last heartbeat time difference (seconds)
  const lastHeartbeatSeconds = useMemo(() => {
    if (!agentNode.lastHeartbeat) return null
    const now = Date.now()
    const lastHeartbeat = new Date(agentNode.lastHeartbeat).getTime()
    return Math.floor((now - lastHeartbeat) / 1000)
  }, [agentNode.lastHeartbeat])

  // Determine whether the heartbeat has expired (more than 30 seconds)
  const isHeartbeatStale = lastHeartbeatSeconds !== null && lastHeartbeatSeconds > 30

  const { display: heartbeatDisplay, title: heartbeatTitle } = formatHeartbeatTime(agentNode.lastHeartbeat)

  return (
    <div
      className={cn(
        "group relative rounded-lg border bg-card transition-[background-color,border-color,box-shadow,opacity] duration-200",
        "hover:shadow-md hover:border-primary/20",
        getAgentRuntimeBorderClass(agentNode.status),
        getAgentRuntimeShellClass(agentNode.status)
      )}
    >
      {/* First line: basic information */}
      <div className="flex gap-3 items-center p-3 pb-2">
        <div className="flex flex-1 gap-2 items-center min-w-0">
          <Status status={getAgentRuntimeStatus(agentNode.status)}>
            <StatusIndicator />
          </Status>

          <div className="flex flex-1 gap-2 items-center min-w-0">
            <span className={cn("truncate", textRole.tableCellPrimary)}>{agentNode.displayName}</span>
            {agentNode.agentVersion && (
              <Badge variant="secondary" className="shrink-0 text-[10px]">
                {agentNode.agentVersion}
              </Badge>
            )}
          </div>

          <Badge
            variant={getAgentHealthBadgeVariant(healthState)}
            className={cn("text-[10px] shrink-0", getAgentHealthClassName(healthState))}
          >
            {healthLabel}
          </Badge>

          {hasWarnings && (
            <Badge variant="warning" className="shrink-0 text-[10px]">
              <IconAlertTriangle className="h-3 mr-1 w-3" />
              {t("card.warning")}
            </Badge>
          )}
        </div>

        <div className="flex gap-2 items-center shrink-0">
          <div className="gap-1 hidden items-center sm:flex text-muted-foreground text-xs">
            <span>{agentNode.observedHostname || t("unknownHost")}</span>
            <span>·</span>
            <span>{connectionIpDisplay}</span>
          </div>

          <div className="flex gap-1 items-center text-muted-foreground text-xs">
            <span className="hidden md:inline">{t("metrics.lastHeartbeat")}:</span>
            <span
              className={cn("font-medium", getAgentHeartbeatTextClass())}
              title={heartbeatTitle}
            >
              {heartbeatDisplay}
            </span>
            {agentNode.status === "online" && !isHeartbeatStale && (
              <IconActivity className={cn("h-3 w-3", getAgentRuntimeTextClass("online"))} />
            )}
          </div>

          <DenseRowActionMenu
            ariaLabel={t("actions.title")}
            ownerClassName="shrink-0"
          >
            <DropdownMenuLabel>{t("actions.title")}</DropdownMenuLabel>
            <DropdownMenuItem onClick={() => onConfig(agentNode)}>
              <IconSettings className="h-4 w-4" />
              {t("actions.config")}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem variant="destructive" onClick={() => onDelete(agentNode)}>
              <IconTrash className="h-4 w-4" />
              {t("actions.delete")}
            </DropdownMenuItem>
          </DenseRowActionMenu>
        </div>
      </div>

      {/* Second row: indicators */}
      {heartbeat ? (
        <div className="flex flex-wrap gap-4 items-center pb-3 pt-1 px-3">
          <MetricBar
            label={t("metrics.cpu")}
            value={heartbeat.cpu}
            threshold={agentNode.cpuThreshold}
          />
          <MetricBar
            label={t("metrics.mem")}
            value={heartbeat.mem}
            threshold={agentNode.memThreshold}
          />
          <MetricBar
            label={t("metrics.disk")}
            value={heartbeat.disk}
            threshold={agentNode.diskThreshold}
          />

          <div className="flex gap-1.5 items-center ml-auto">
            <span className="text-[10px] text-muted-foreground">{t("metrics.runningTasks")}:</span>
            <span className="font-medium text-[10px]">
              {formatNumber.formatInteger(heartbeat.runningTasks)}
            </span>
          </div>

          <div className="flex gap-1.5 items-center">
            <span className="text-[10px] text-muted-foreground">{t("metrics.usedTaskSlots")}:</span>
            <span className="font-medium text-[10px]">
              {formatNumber.formatInteger(heartbeat.taskSlotsUsed)}/{agentNode.maxTasks}
            </span>
          </div>

          <div className="flex gap-1.5 items-center">
            <span className="text-[10px] text-muted-foreground">{t("list.uptime")}:</span>
            <span className="font-medium text-[10px]">{formatUptime(heartbeat.uptime)}</span>
          </div>
        </div>
      ) : (
        <div className="pb-3 pt-1 px-3">
          <div className="bg-muted/20 border border-dashed py-2 rounded text-center text-muted-foreground text-xs">
            {t("card.waitingForHeartbeat")}
          </div>
        </div>
      )}
    </div>
  )
}
