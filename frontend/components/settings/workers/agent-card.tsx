"use client"

import { useMemo, useState } from "react"
import { useTranslations } from "next-intl"
import {
  IconDotsVertical,
  IconSettings,
  IconTrash,
  IconChevronDown,
  IconChevronUp,
  IconAlertTriangle,
  IconActivity,
} from "@/components/icons"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import { Status, StatusIndicator, StatusLabel } from "@/components/ui/shadcn-io/status"
import { useFormatNumber, useFormatRelativeTime } from "@/lib/i18n-format"
import { cn } from "@/lib/utils"
import type { Agent } from "@/types/agent.types"
import { MetricProgress } from "./metric-progress"

function getHealthStyle(state: string) {
  const normalized = state.toLowerCase()
  if (normalized === "ok") {
    return "bg-[var(--success)]/10 text-[var(--success)] border-[var(--success)]/20"
  }
  if (normalized === "warning" || normalized === "warn") {
    return "bg-[var(--warning)]/10 text-[var(--warning)] border-[var(--warning)]/20"
  }
  if (normalized === "error" || normalized === "critical") {
    return "bg-[var(--error)]/10 text-[var(--error)] border-[var(--error)]/20"
  }
  return "bg-muted text-muted-foreground border-border"
}

function getStatusVariant(status: string) {
  if (status === "online") return "online"
  if (status === "offline") return "offline"
  return "maintenance"
}

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

interface AgentCardProps {
  agent: Agent
  onConfig: (agent: Agent) => void
  onDelete: (agent: Agent) => void
}

export function AgentCard({
  agent,
  onConfig,
  onDelete,
}: AgentCardProps) {
  const t = useTranslations("settings.workers")
  const formatRelativeTime = useFormatRelativeTime()
  const formatNumber = useFormatNumber()
  const [isExpanded, setIsExpanded] = useState(false)

  const statusLabel = useMemo(() => {
    if (agent.status === "online") return t("status.online")
    if (agent.status === "offline") return t("status.offline")
    return t("status.unknown")
  }, [agent.status, t])

  const healthState = (agent.health?.state || "unknown").toLowerCase()
  const healthLabel = useMemo(() => {
    if (healthState === "ok") return t("health.ok")
    if (healthState === "warning" || healthState === "warn") return t("health.warning")
    if (healthState === "error" || healthState === "critical") return t("health.error")
    return t("health.unknown")
  }, [healthState, t])

  const heartbeat = agent.heartbeat

  // Check if any metric exceeds the threshold
  const hasWarnings = useMemo(() => {
    if (!heartbeat) return false
    return (
      heartbeat.cpu >= agent.cpuThreshold ||
      heartbeat.mem >= agent.memThreshold ||
      heartbeat.disk >= agent.diskThreshold
    )
  }, [heartbeat, agent])

  // Calculate last heartbeat time difference (seconds)
  const lastHeartbeatSeconds = useMemo(() => {
    if (!agent.lastHeartbeat) return null
    const now = Date.now()
    const lastHeartbeat = new Date(agent.lastHeartbeat).getTime()
    return Math.floor((now - lastHeartbeat) / 1000)
  }, [agent.lastHeartbeat])

  // Determine whether the heartbeat has expired (more than 30 seconds)
  const isHeartbeatStale = lastHeartbeatSeconds !== null && lastHeartbeatSeconds > 30

  return (
    <Card
      className={cn(
        "transition-[background-color,border-color,box-shadow,opacity] duration-200 hover:shadow-md",
        agent.status === "online" && "border-[var(--success)]/20",
        agent.status === "offline" && "border-slate-300/50 opacity-75"
      )}
    >
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0 flex-1 space-y-2">
            <div className="flex flex-wrap items-center gap-2">
              <CardTitle className="text-base truncate">{agent.name}</CardTitle>
              {agent.version && (
                <Badge variant="secondary" className="text-[10px] shrink-0">
                  {agent.version}
                </Badge>
              )}
            </div>

            <div className="flex flex-wrap items-center gap-2">
              <Status status={getStatusVariant(agent.status)}>
                <StatusIndicator className={agent.status === "online" ? "animate-pulse" : ""} />
                <StatusLabel>{statusLabel}</StatusLabel>
              </Status>
              <Badge variant="outline" className={getHealthStyle(healthState)}>
                {healthLabel}
              </Badge>
              {hasWarnings && (
                <Badge variant="outline" className="bg-[var(--warning)]/10 text-[var(--warning)] border-[var(--warning)]/20">
                  <IconAlertTriangle className="h-3 w-3 mr-1" />
                  {t("card.warning")}
                </Badge>
              )}
            </div>

            <div className="text-xs text-muted-foreground flex items-center gap-1">
              <span className="truncate">
                {agent.hostname || t("unknownHost")} · {agent.ipAddress || t("unknownIp")}
              </span>
            </div>
          </div>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="h-8 w-8 shrink-0" aria-label={t("actions.title")}>
                <IconDotsVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel>{t("actions.title")}</DropdownMenuLabel>
              <DropdownMenuItem onClick={() => onConfig(agent)}>
                <IconSettings className="h-4 w-4" />
                {t("actions.config")}
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem variant="destructive" onClick={() => onDelete(agent)}>
                <IconTrash className="h-4 w-4" />
                {t("actions.delete")}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardHeader>

      <CardContent className="space-y-3">
        {/* Basic information - always shown */}
        <div className="grid grid-cols-2 gap-2 text-xs">
          <div className="rounded-lg border bg-muted/40 p-2.5">
            <p className="text-muted-foreground mb-1">{t("metrics.lastHeartbeat")}</p>
            <div className="flex items-center gap-1">
              <p className={cn("font-medium", isHeartbeatStale && "text-[var(--warning)]")}>
                {formatRelativeTime(agent.lastHeartbeat)}
              </p>
              {agent.status === "online" && !isHeartbeatStale && (
                <IconActivity className="h-3 w-3 text-[var(--success)] animate-pulse" />
              )}
            </div>
          </div>
          <div className="rounded-lg border bg-muted/40 p-2.5">
            <p className="text-muted-foreground mb-1">{t("metrics.connectedAt")}</p>
            <p className="font-medium">{formatRelativeTime(agent.connectedAt)}</p>
          </div>
        </div>

        {/* Collapsible details */}
        {heartbeat && (
          <Collapsible open={isExpanded} onOpenChange={setIsExpanded}>
            <CollapsibleTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                className="w-full justify-between text-xs h-8"
              >
                <span>{isExpanded ? t("card.hideDetails") : t("card.showDetails")}</span>
                {isExpanded ? (
                  <IconChevronUp className="h-4 w-4" />
                ) : (
                  <IconChevronDown className="h-4 w-4" />
                )}
              </Button>
            </CollapsibleTrigger>

            <CollapsibleContent className="space-y-3 pt-3">
              {/* System Metrics - with progress bar and threshold warnings */}
              <div className="space-y-2.5 rounded-lg border bg-muted/20 p-3">
                <p className="text-xs font-medium text-muted-foreground mb-2">
                  {t("card.systemMetrics")}
                </p>
                <MetricProgress
                  label={t("metrics.cpu")}
                  value={heartbeat.cpu}
                  threshold={agent.cpuThreshold}
                />
                <MetricProgress
                  label={t("metrics.mem")}
                  value={heartbeat.mem}
                  threshold={agent.memThreshold}
                />
                <MetricProgress
                  label={t("metrics.disk")}
                  value={heartbeat.disk}
                  threshold={agent.diskThreshold}
                />
              </div>

              {/* Tasks and run times */}
              <div className="grid grid-cols-2 gap-2 text-xs">
                <div className="rounded-lg border bg-muted/30 p-2.5">
                  <p className="text-muted-foreground mb-1">{t("metrics.tasks")}</p>
                  <p className="font-medium text-lg">
                    {formatNumber.formatInteger(heartbeat.tasks)}
                    <span className="text-xs text-muted-foreground ml-1">
                      / {agent.maxTasks}
                    </span>
                  </p>
                </div>
                <div className="rounded-lg border bg-muted/30 p-2.5">
                  <p className="text-muted-foreground mb-1">{t("metrics.uptime")}</p>
                  <p className="font-medium text-lg">{formatUptime(heartbeat.uptime)}</p>
                </div>
              </div>

              {/* Configuration information */}
              <div className="rounded-lg border bg-muted/20 p-3">
                <p className="text-xs font-medium text-muted-foreground mb-2">
                  {t("card.configuration")}
                </p>
                <div className="flex flex-wrap gap-1.5">
                  <Badge variant="outline" className="text-[10px]">
                    {t("metrics.maxTasks", { value: agent.maxTasks })}
                  </Badge>
                  <Badge variant="outline" className="text-[10px]">
                    CPU: {agent.cpuThreshold}%
                  </Badge>
                  <Badge variant="outline" className="text-[10px]">
                    MEM: {agent.memThreshold}%
                  </Badge>
                  <Badge variant="outline" className="text-[10px]">
                    DISK: {agent.diskThreshold}%
                  </Badge>
                </div>
              </div>

              {/* Last updated */}
              {heartbeat.updatedAt && (
                <div className="text-[10px] text-muted-foreground text-center pt-1">
                  {t("metrics.updatedAt", { value: formatRelativeTime(heartbeat.updatedAt) })}
                </div>
              )}
            </CollapsibleContent>
          </Collapsible>
        )}

        {/* Prompt when there is no heartbeat data */}
        {!heartbeat && (
          <div className="rounded-lg border border-dashed bg-muted/20 p-4 text-center">
            <p className="text-xs text-muted-foreground">
              {t("card.waitingForHeartbeat")}
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
