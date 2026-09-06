import {
  getRuntimeStatusTone,
  getStatusToneBadgeClass,
  getStatusToneBgClass,
  getStatusToneTextClass,
  type RuntimeStatus,
  type StatusBadgeVariant,
  type StatusTone,
} from "@/lib/status-config"
import {
  getAgentHealthDistributionStatus,
  getAgentStatusDistributionTone,
  type AgentStatusDistributionStatus,
} from "@/lib/agent-status-distribution"

export type { AgentStatusDistributionStatus } from "@/lib/agent-status-distribution"
export { getAgentStatusDistributionTone } from "@/lib/agent-status-distribution"

export type AgentMetricStatus = "normal" | "warning" | "critical"

export function getAgentRuntimeStatus(status: string): RuntimeStatus {
  if (status === "online") return "online"
  if (status === "offline") return "offline"
  return "maintenance"
}

export function getAgentRuntimeBorderClass(status: string): string {
  return status === "online" ? "border-success/20" : "border-border/80"
}

export function getAgentRuntimeShellClass(status: string): string {
  return getAgentRuntimeStatus(status) === "offline" ? "border-border/80 opacity-75" : ""
}

export function getAgentRuntimeTextClass(status: string): string {
  return getStatusToneTextClass(getRuntimeStatusTone(getAgentRuntimeStatus(status)))
}

export function getAgentHeartbeatTextClass(): string {
  return "text-foreground"
}

export function getAgentHealthTone(state: string): StatusTone {
  const status = getAgentHealthDistributionStatus(state)
  if (status === "healthy") return "success"
  if (status === "warning") return "warning"
  return "muted"
}

export function getAgentHealthBadgeVariant(state: string): StatusBadgeVariant {
  const tone = getAgentHealthTone(state)
  if (tone === "success") return "success"
  if (tone === "warning") return "warning"
  if (tone === "error") return "error"
  return "outline"
}

export function getAgentHealthClassName(state: string): string {
  const tone = getAgentHealthTone(state)
  if (tone === "muted") return "bg-muted text-muted-foreground border-border"
  return getStatusToneBadgeClass(tone)
}

export function getAgentStatusStripBarClass(status: string, healthState?: string | null): string {
  const runtimeStatus = getAgentRuntimeStatus(status)
  if (runtimeStatus === "offline") return "bg-error/80"

  const tone = getAgentHealthTone(healthState || "ok")
  if (tone === "warning" || tone === "error") return "bg-warning/80"
  return "bg-success/80"
}

export function getAgentStatusDistributionBarClass(status: AgentStatusDistributionStatus): string {
  return getStatusToneBgClass(getAgentStatusDistributionTone(status))
}

export function getAgentStatusDistributionDotClass(status: AgentStatusDistributionStatus): string {
  return getStatusToneBgClass(getAgentStatusDistributionTone(status))
}

export function getAgentMetricStatus(percentage: number, threshold?: number): AgentMetricStatus {
  if (!threshold) return "normal"
  if (percentage >= threshold) return "critical"
  if (percentage >= threshold * 0.8) return "warning"
  return "normal"
}

export function getAgentMetricTone(percentage: number, threshold?: number): StatusTone {
  const status = getAgentMetricStatus(percentage, threshold)
  if (status === "critical") return "error"
  if (status === "warning") return "warning"
  return "success"
}

export function getAgentMetricBarClass(percentage: number, threshold?: number): string {
  return getStatusToneBgClass(getAgentMetricTone(percentage, threshold))
}

export function getAgentMetricTextClass(percentage: number, threshold?: number): string {
  const tone = getAgentMetricTone(percentage, threshold)
  return tone === "success" ? "text-foreground" : getStatusToneTextClass(tone)
}

export function getAgentMetricAlertTextClass(percentage: number, threshold?: number): string {
  return getStatusToneTextClass(getAgentMetricTone(percentage, threshold))
}

export function getAgentMetricTrackClass(percentage: number, threshold?: number): string {
  const status = getAgentMetricStatus(percentage, threshold)
  if (status === "critical") return "bg-error/20"
  if (status === "warning") return "bg-warning/20"
  return ""
}
