import {
  getAgentStatusDistributionStatus,
  getAgentStatusDistributionTone,
  type AgentStatusDistributionStatus,
  type AgentStatusDistributionTone,
} from "@/lib/agent-status-distribution"
import { getStatusToneColorVar } from "@/lib/status-config"
import type { AgentLocationMapAgent, AgentLocationState } from "@/types/agent.types"

export type AgentLocationMapNode = {
  id: string
  resourceName: string
  name: string
  latitude: number
  longitude: number
  status: AgentStatusDistributionStatus
  tone: AgentStatusDistributionTone
  color: string
  locationState: Exclude<AgentLocationState, "unknown">
  sourceObservedIp: string
  taskSlotsUsed: number
  hasTaskLoadData: boolean
}

export function isValidAgentLocationCoordinate(latitude: number | null | undefined, longitude: number | null | undefined) {
  return (
    typeof latitude === "number" &&
    typeof longitude === "number" &&
    Number.isFinite(latitude) &&
    Number.isFinite(longitude) &&
    latitude >= -90 &&
    latitude <= 90 &&
    longitude >= -180 &&
    longitude <= 180
  )
}

export function buildAgentLocationMapNodes(agents: AgentLocationMapAgent[]): AgentLocationMapNode[] {
  return agents.flatMap((agent) => {
    if (!isValidAgentLocationCoordinate(agent.location.latitude, agent.location.longitude)) {
      return []
    }

    const status = getAgentStatusDistributionStatus({
      status: agent.status,
      health: { state: agent.healthState },
    })
    const tone = getAgentStatusDistributionTone(status)

    return [{
      id: `agent-location-${agent.id}`,
      resourceName: agent.resourceName,
      name: agent.displayName.trim() || agent.resourceName,
      latitude: agent.location.latitude,
      longitude: agent.location.longitude,
      status,
      tone,
      color: getStatusToneColorVar(tone),
      locationState: agent.location.state,
      sourceObservedIp: agent.location.sourceObservedIp,
      taskSlotsUsed: agent.taskSlotsUsed ?? 0,
      hasTaskLoadData: agent.taskSlotsUsed !== null,
    }]
  })
}
