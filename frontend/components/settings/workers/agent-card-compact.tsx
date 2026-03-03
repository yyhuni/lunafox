"use client"

import { type ReactNode, useMemo } from "react"
import { useNow, useTranslations } from "next-intl"
import {
  IconDotsVertical,
  IconSettings,
  IconTrash,
  IconActivity,
  IconClock,
  IconCpu,
  IconDatabase,
  IconTerminal,
  HardDrive,
} from "@/components/icons"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card } from "@/components/ui/card"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useFormatNumber, useFormatRelativeTime } from "@/lib/i18n-format"
import { cn } from "@/lib/utils"
import type { Agent } from "@/types/agent.types"

function getStatusVariant(status: string) {
  const normalized = status.toLowerCase()
  if (normalized === "online") return "online"
  if (normalized === "offline") return "offline"
  if (normalized === "maintenance") return "maintenance"
  return "unknown"
}

function StatusBadge({ status, label }: { status: string; label: string }) {
  const variant = getStatusVariant(status)
  const styles = {
    online: "bg-[var(--success)]/15 text-[var(--success)] border-[var(--success)]/25",
    offline: "bg-muted text-muted-foreground border-border",
    maintenance: "bg-[var(--warning)]/15 text-[var(--warning)] border-[var(--warning)]/25",
    unknown: "bg-muted text-muted-foreground border-border",
  }

  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center rounded-full border px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide",
        styles[variant]
      )}
    >
      {label}
    </span>
  )
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

interface MetricSegmentedProgressProps {
  label: string
  value: number
  threshold?: number
  icon?: ReactNode
}

function MetricSegmentedProgress({ label, value, threshold, icon }: MetricSegmentedProgressProps) {
  const percentage = Math.min(100, Math.max(0, value))
  const segmentCount = 25

  const isCritical = threshold !== undefined && percentage >= threshold
  const activeColorClass = isCritical ? "bg-[var(--error)]" : "bg-[var(--success)]"

  const activeCount = Math.ceil((percentage / 100) * segmentCount)
  const thresholdValue = threshold === undefined ? undefined : Math.min(100, Math.max(0, threshold))
  const thresholdIndex = thresholdValue === undefined
    ? undefined
    : Math.floor((thresholdValue / 100) * segmentCount)

  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between text-[10px]">
        <div className="flex items-center gap-1.5">
          {icon && <span className="text-muted-foreground/80">{icon}</span>}
          <span>{label}</span>
        </div>
        <div className="flex items-center gap-1.5">
          <span className={cn("tabular-nums", isCritical && "text-[var(--error)]")}>
            {percentage.toFixed(0)}%
          </span>
          {thresholdValue !== undefined && (
            <span className="text-muted-foreground/70">/ {thresholdValue}%</span>
          )}
        </div>
      </div>

      <div className="flex gap-[2px] h-1.5 w-full">
        {Array.from({ length: segmentCount }).map((_, index) => {
          const isActive = index < activeCount
          const isThresholdMarker = thresholdIndex !== undefined && index === thresholdIndex

          return (
            <div
              key={index}
              className={cn(
                "flex-1 rounded-[1px]",
                isActive
                  ? activeColorClass
                  : isThresholdMarker
                    ? "bg-foreground/30"
                    : "bg-muted"
              )}
            />
          )
        })}
      </div>
    </div>
  )
}

interface AgentCardCompactProps {
  agent: Agent
  onConfig: (agent: Agent) => void
  onDelete: (agent: Agent) => void
  onLogs: (agent: Agent) => void
}

function normalizePastDate(value: string | Date | null | undefined, nowMs: number): Date | null {
  if (!value) return null
  const parsed = value instanceof Date ? value : new Date(value)
  const parsedMs = parsed.getTime()
  if (Number.isNaN(parsedMs)) return null
  return new Date(Math.min(parsedMs, nowMs))
}

export function AgentCardCompact({
  agent,
  onConfig,
  onDelete,
  onLogs,
}: AgentCardCompactProps) {
  const t = useTranslations("settings.workers")
  const formatRelativeTime = useFormatRelativeTime()
  const now = useNow({ updateInterval: 5000 })
  const formatNumber = useFormatNumber()

  const heartbeat = agent.heartbeat
  const nowMs = now.getTime()
  const lastSeenDate = useMemo(() => normalizePastDate(agent.lastHeartbeat, nowMs), [agent.lastHeartbeat, nowMs])

  const lastHeartbeatSeconds = useMemo(() => {
    if (!lastSeenDate) return null
    return Math.floor((nowMs - lastSeenDate.getTime()) / 1000)
  }, [lastSeenDate, nowMs])

  const isHeartbeatStale = lastHeartbeatSeconds !== null && lastHeartbeatSeconds > 30
  const lastSeenText = formatRelativeTime(lastSeenDate)
  const statusVariant = getStatusVariant(agent.status)
  const statusLabel = statusVariant === "online"
    ? t("status.online")
    : statusVariant === "offline"
      ? t("status.offline")
      : statusVariant === "maintenance"
        ? t("status.maintenance")
        : t("status.unknown")

  return (
    <Card className="group flex flex-col rounded-lg border border-border bg-card text-card-foreground shadow-sm hover:shadow-md transition-[background-color,border-color,box-shadow,opacity] duration-200 overflow-hidden gap-0 py-0">
      <div className="flex items-center justify-between p-3 border-b border-border/50 bg-muted/20">
        <div className="min-w-0 flex flex-1 items-center gap-2.5">
          <StatusBadge status={agent.status} label={statusLabel} />
          <div className="min-w-0">
            <div className="font-bold text-sm leading-none truncate" title={agent.name}>
              {agent.name}
            </div>
            <div className="text-[10px] text-muted-foreground font-mono mt-1 opacity-80 truncate" title={agent.ipAddress || t("unknownIp")}>
              {agent.ipAddress || t("unknownIp")}
            </div>
          </div>
        </div>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 text-muted-foreground hover:text-foreground shrink-0"
              aria-label={t("actions.title")}
            >
              <IconDotsVertical className="w-4 h-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuLabel>{t("actions.title")}</DropdownMenuLabel>
            <DropdownMenuItem onClick={() => onConfig(agent)}>
              <IconSettings className="w-4 h-4" />
              {t("actions.config")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => onLogs(agent)}>
              <IconTerminal className="w-4 h-4" />
              {t("actions.logs")}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem variant="destructive" onClick={() => onDelete(agent)}>
              <IconTrash className="w-4 h-4" />
              {t("actions.delete")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <div className="p-4 space-y-4 flex-1">
        <div className="grid grid-cols-2 gap-x-4 gap-y-2 text-[11px]">
          <div className="space-y-0.5 min-w-0">
            <span className="text-[9px] uppercase text-muted-foreground font-bold tracking-wider">Host</span>
            <div className="font-mono text-xs truncate" title={agent.hostname || t("unknownHost")}>
              {agent.hostname || t("unknownHost")}
            </div>
          </div>

          <div className="space-y-0.5 text-right min-w-0">
            <span className="text-[9px] uppercase text-muted-foreground font-bold tracking-wider">Version</span>
            <div className="font-mono text-xs truncate" title={agent.agentVersion || "-"}>
              {agent.agentVersion || "-"}
            </div>
          </div>
        </div>

        {heartbeat ? (
          <div className="space-y-3 pt-2">
            <MetricSegmentedProgress
              label={t("metrics.cpu")}
              value={heartbeat.cpu}
              threshold={agent.cpuThreshold}
              icon={<IconCpu className="h-3 w-3" />}
            />
            <MetricSegmentedProgress
              label={t("metrics.mem")}
              value={heartbeat.mem}
              threshold={agent.memThreshold}
              icon={<IconDatabase className="h-3 w-3" />}
            />
            <MetricSegmentedProgress
              label={t("metrics.disk")}
              value={heartbeat.disk}
              threshold={agent.diskThreshold}
              icon={<HardDrive className="h-3 w-3" />}
            />
          </div>
        ) : (
          <div className="text-xs text-muted-foreground text-center py-6 border border-dashed border-border rounded bg-muted/20">
            {t("card.waitingForHeartbeat")}
          </div>
        )}
      </div>

      <div className="p-3 border-t border-border/50 bg-muted/5 flex items-center justify-between text-[10px] gap-2">
        <div className="flex items-center gap-3 text-muted-foreground min-w-0">
          <div className="flex items-center gap-1.5 min-w-0" title={t("metrics.lastHeartbeat")}>
            <IconActivity className="w-3 h-3 shrink-0" />
            <span className={cn(
              "truncate",
              agent.status === "offline"
                ? "text-[var(--error)]"
                : isHeartbeatStale
                  ? "text-[var(--warning)]"
                  : undefined
            )}>
              {lastSeenText}
            </span>
          </div>
          <div className="flex items-center gap-1.5 shrink-0" title={t("list.uptime")}>
            <IconClock className="w-3 h-3" />
            <span>{heartbeat ? formatUptime(heartbeat.uptime) : "-"}</span>
          </div>
        </div>

        <div className="flex items-center gap-1.5 shrink-0">
          <span className="text-[9px] uppercase font-bold text-muted-foreground">{t("metrics.tasks")}</span>
          <Badge variant="secondary" className="h-5 text-[10px] font-mono px-1.5 bg-muted border-border text-foreground">
            {heartbeat ? formatNumber.formatInteger(heartbeat.tasks) : "-"}
            <span className="opacity-40 mx-0.5">/</span>
            {agent.maxTasks}
          </Badge>
        </div>
      </div>
    </Card>
  )
}
