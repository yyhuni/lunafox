import type { Agent } from "@/types/agent.types"

export type AgentConnectionIpDisplayLabels = {
  offline: string
  unobserved: string
}

// Management views must never substitute GeoIP provenance for a live control connection.
export function getAgentConnectionIpDisplay(
  agent: Pick<Agent, "connectionIp" | "status">,
  labels: AgentConnectionIpDisplayLabels,
): string {
  if (agent.connectionIp) {
    return agent.connectionIp
  }

  return agent.status === "offline" ? labels.offline : labels.unobserved
}
