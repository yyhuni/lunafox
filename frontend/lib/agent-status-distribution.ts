import type { Agent } from "@/types/agent.types"

export const AGENT_HEALTHY_HEALTH_STATES = ["ok", "healthy"] as const
export const AGENT_WARNING_HEALTH_STATES = ["warning", "warn", "error", "critical"] as const

export type AgentStatusDistributionStatus = "healthy" | "warning" | "offline" | "unknown"
export type AgentStatusDistributionTone = "success" | "warning" | "error" | "muted"

export type AgentStatusDistribution = Record<AgentStatusDistributionStatus, number>

function isOneOf<T extends readonly string[]>(value: string, options: T): value is T[number] {
  return (options as readonly string[]).includes(value)
}

export function getAgentHealthDistributionStatus(
  healthState: string | null | undefined,
): Exclude<AgentStatusDistributionStatus, "offline"> {
  const normalized = healthState?.toLowerCase() ?? ""

  if (isOneOf(normalized, AGENT_HEALTHY_HEALTH_STATES)) return "healthy"
  if (isOneOf(normalized, AGENT_WARNING_HEALTH_STATES)) return "warning"

  return "unknown"
}

export function getAgentStatusDistributionStatus(
  agentNode: Pick<Agent, "status" | "health">,
): AgentStatusDistributionStatus {
  // This order matches node management: transport state wins over health metadata.
  if (agentNode.status === "offline") return "offline"
  if (agentNode.status !== "online") return "unknown"

  return getAgentHealthDistributionStatus(agentNode.health?.state)
}

export function getAgentStatusDistributionTone(
  status: AgentStatusDistributionStatus,
): AgentStatusDistributionTone {
  if (status === "healthy") return "success"
  if (status === "warning") return "warning"
  if (status === "offline") return "error"
  return "muted"
}

export function buildAgentStatusDistribution(
  nodes: Agent[],
  total = nodes.length,
): AgentStatusDistribution {
  const distribution = nodes.reduce<AgentStatusDistribution>((counts, agentNode) => {
    counts[getAgentStatusDistributionStatus(agentNode)] += 1
    return counts
  }, {
    healthy: 0,
    warning: 0,
    offline: 0,
    unknown: 0,
  })

  // The source page's query can represent more nodes than its summary page size.
  distribution.unknown = Math.max(total - distribution.healthy - distribution.warning - distribution.offline, 0)

  return distribution
}
