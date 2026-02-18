"use client"

import { useMemo } from "react"
import { useTranslations } from "next-intl"
import {
  IconDotsVertical,
  IconSettings,
  IconTrash,
  IconAlertTriangle,
  IconActivity,
} from "@/components/icons"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Status, StatusIndicator } from "@/components/ui/shadcn-io/status"
import { useFormatNumber, useFormatRelativeTime } from "@/lib/i18n-format"
import { cn } from "@/lib/utils"
import type { Agent } from "@/types/agent.types"

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

interface MetricBarProps {
  label: string
  value: number
  threshold?: number
  className?: string
}

function MetricBar({ label, value, threshold, className }: MetricBarProps) {
  const percentage = Math.min(100, Math.max(0, value))

  const status = useMemo(() => {
    if (!threshold) return "normal"
    if (percentage >= threshold) return "critical"
    if (percentage >= threshold * 0.8) return "warning"
    return "normal"
  }, [percentage, threshold])

  const progressColor = useMemo(() => {
    if (status === "critical") return "bg-[var(--error)]"
    if (status === "warning") return "bg-[var(--warning)]"
    return "bg-[var(--success)]"
  }, [status])

  return (
    <div className={cn("flex items-center gap-1.5", className)}>
      <span className="text-[10px] text-muted-foreground min-w-[32px]">{label}</span>
      <div className="relative h-1.5 w-16 bg-muted rounded-full overflow-hidden">
        <div
          className={cn("h-full transition-[width,background-color] duration-300", progressColor)}
          style={{ width: `${percentage}%` }}
        />
      </div>
      <span className={cn(
        "text-[10px] font-medium tabular-nums text-right",
        status === "critical" && "text-[var(--error)]",
        status === "warning" && "text-[var(--warning)]"
      )}>
        {percentage.toFixed(0)}%
        {threshold && (
          <span className="text-muted-foreground">/{threshold}%</span>
        )}
      </span>
      {status !== "normal" && (
        <IconAlertTriangle className="h-3 w-3 text-[var(--warning)] shrink-0" />
      )}
    </div>
  )
}

interface AgentListItemProps {
  agent: Agent
  onConfig: (agent: Agent) => void
  onDelete: (agent: Agent) => void
}

export function AgentListItem({
  agent,
  onConfig,
  onDelete,
}: AgentListItemProps) {
  const t = useTranslations("settings.workers")
  const formatRelativeTime = useFormatRelativeTime()
  const formatNumber = useFormatNumber()


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
    <div
      className={cn(
        "group relative rounded-lg border bg-card transition-[background-color,border-color,box-shadow,opacity] duration-200",
        "hover:shadow-md hover:border-primary/20",
        agent.status === "online" && "border-[var(--success)]/20",
        agent.status === "offline" && "border-slate-300/50 opacity-75"
      )}
    >
      {/* First line: basic information */}
      <div className="flex items-center gap-3 p-3 pb-2">
        <div className="flex items-center gap-2 min-w-0 flex-1">
          <Status status={getStatusVariant(agent.status)}>
            <StatusIndicator className={agent.status === "online" ? "animate-pulse" : ""} />
          </Status>

          <div className="flex items-center gap-2 min-w-0 flex-1">
            <span className="font-medium text-sm truncate">{agent.name}</span>
            {agent.version && (
              <Badge variant="secondary" className="text-[10px] shrink-0">
                {agent.version}
              </Badge>
            )}
          </div>

          <Badge variant="outline" className={cn("text-[10px] shrink-0", getHealthStyle(healthState))}>
            {healthLabel}
          </Badge>

          {hasWarnings && (
            <Badge variant="outline" className="bg-[var(--warning)]/10 text-[var(--warning)] border-[var(--warning)]/20 text-[10px] shrink-0">
              <IconAlertTriangle className="h-3 w-3 mr-1" />
              {t("card.warning")}
            </Badge>
          )}
        </div>

        <div className="flex items-center gap-2 shrink-0">
          <div className="text-xs text-muted-foreground hidden sm:flex items-center gap-1">
            <span>{agent.hostname || t("unknownHost")}</span>
            <span>·</span>
            <span>{agent.ipAddress || t("unknownIp")}</span>
          </div>

          <div className="text-xs text-muted-foreground flex items-center gap-1">
            <span className="hidden md:inline">{t("metrics.lastHeartbeat")}:</span>
            <span className={cn("font-medium", isHeartbeatStale && "text-[var(--warning)]")}>
              {formatRelativeTime(agent.lastHeartbeat)}
            </span>
            {agent.status === "online" && !isHeartbeatStale && (
              <IconActivity className="h-3 w-3 text-[var(--success)] animate-pulse" />
            )}
          </div>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="h-7 w-7 shrink-0" aria-label={t("actions.title")}>
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
      </div>

      {/* Second row: indicators */}
      {heartbeat ? (
        <div className="flex items-center gap-4 px-3 pb-3 pt-1 flex-wrap">
          <MetricBar
            label={t("metrics.cpu")}
            value={heartbeat.cpu}
            threshold={agent.cpuThreshold}
          />
          <MetricBar
            label={t("metrics.mem")}
            value={heartbeat.mem}
            threshold={agent.memThreshold}
          />
          <MetricBar
            label={t("metrics.disk")}
            value={heartbeat.disk}
            threshold={agent.diskThreshold}
          />

          <div className="flex items-center gap-1.5 ml-auto">
            <span className="text-[10px] text-muted-foreground">{t("metrics.tasks")}:</span>
            <span className="text-[10px] font-medium">
              {formatNumber.formatInteger(heartbeat.tasks)}/{agent.maxTasks}
            </span>
          </div>

          <div className="flex items-center gap-1.5">
            <span className="text-[10px] text-muted-foreground">{t("list.uptime")}:</span>
            <span className="text-[10px] font-medium">{formatUptime(heartbeat.uptime)}</span>
          </div>
        </div>
      ) : (
        <div className="px-3 pb-3 pt-1">
          <div className="text-xs text-muted-foreground text-center py-2 border border-dashed rounded bg-muted/20">
            {t("card.waitingForHeartbeat")}
          </div>
        </div>
      )}
    </div>
  )
}
