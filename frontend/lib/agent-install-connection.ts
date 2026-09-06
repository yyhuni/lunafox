import type { Agent } from "@/types/agent.types"

export type AgentInstallConnectionPhase = "waiting" | "registered" | "online"
export type DetectedAgentInstallStatus = "registered" | "online"

export interface DetectedInstallAgent {
  id: number
  name: string
  address: string
  status: DetectedAgentInstallStatus
  createdAt: string
}

export interface AgentInstallConnectionState {
  phase: AgentInstallConnectionPhase
  agents: DetectedInstallAgent[]
}

export const EMPTY_AGENT_INSTALL_CONNECTION_STATE: AgentInstallConnectionState = {
  phase: "waiting",
  agents: [],
}

export function projectAgentInstallConnection(
  currentAgents: readonly Agent[],
): AgentInstallConnectionState {
  const agents = currentAgents.map(toDetectedInstallAgent).sort(compareDetectedAgents)
  if (agents.length === 0) return EMPTY_AGENT_INSTALL_CONNECTION_STATE

  return {
    phase: agents.every((agent) => agent.status === "online") ? "online" : "registered",
    agents,
  }
}

function toDetectedInstallAgent(agent: Agent): DetectedInstallAgent {
  return {
    id: agent.id,
    name: agent.displayName || agent.observedHostname || agent.name,
    address: agent.connectionIp ?? "",
    status: agent.status === "online" ? "online" : "registered",
    createdAt: agent.createdAt,
  }
}

function compareDetectedAgents(left: DetectedInstallAgent, right: DetectedInstallAgent): number {
  return left.createdAt.localeCompare(right.createdAt) || left.id - right.id
}
