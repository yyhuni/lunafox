import type { AgentExecutionCapacity } from "@/lib/agent-execution-capacity"

export type AgentClusterHealthState = "healthy" | "needsAttention" | "critical" | "unknown"

export type AgentClusterHealthStats = {
  total: number
  online: number
  offline: number
  warning: number
  healthy: number
  unknown: number
}

export type AgentClusterHealthSummary = {
  state: AgentClusterHealthState
  attentionCount: number
  isComplete: boolean
}

type AgentClusterHealthInput = {
  stats: AgentClusterHealthStats
  capacity: AgentExecutionCapacity
  isComplete: boolean
}

/**
 * A paged overview cannot prove the health of the whole cluster. Keep that
 * boundary explicit so a partial first page never appears as a healthy fleet.
 */
export function buildAgentClusterHealthSummary({
  stats,
  capacity,
  isComplete,
}: AgentClusterHealthInput): AgentClusterHealthSummary {
  const attentionCount = stats.offline + stats.warning

  if (!isComplete || stats.total === 0) {
    return { state: "unknown", attentionCount, isComplete }
  }

  if (stats.online === 0 || (capacity.totalSlots > 0 && capacity.availableSlots === 0)) {
    return { state: "critical", attentionCount, isComplete }
  }

  if (
    attentionCount > 0
    || (capacity.totalSlots > 0 && capacity.availableSlots / capacity.totalSlots <= 0.2)
  ) {
    return { state: "needsAttention", attentionCount, isComplete }
  }

  return { state: "healthy", attentionCount, isComplete }
}
