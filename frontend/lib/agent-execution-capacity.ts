import type { Agent } from "@/types/agent.types"

export type AgentExecutionCapacity = {
  totalSlots: number
  occupiedSlots: number
  availableSlots: number
  unavailableSlots: number
}

function nonNegativeInteger(value: number): number {
  return Math.max(0, Math.floor(value))
}

/**
 * Builds the small-cluster overview from the unfiltered Agent result set.
 * An Agent without an online heartbeat cannot receive work, so its configured
 * slots remain visible as unavailable rather than inflating available capacity.
 */
export function buildAgentExecutionCapacity(agents: Agent[]): AgentExecutionCapacity {
  return agents.reduce<AgentExecutionCapacity>((capacity, agent) => {
    const maxTasks = nonNegativeInteger(agent.maxTasks)
    capacity.totalSlots += maxTasks

    if (agent.status !== "online" || !agent.heartbeat) {
      capacity.unavailableSlots += maxTasks
      return capacity
    }

    const occupiedSlots = nonNegativeInteger(agent.heartbeat.taskSlotsUsed)
    capacity.occupiedSlots += occupiedSlots
    capacity.availableSlots += Math.max(maxTasks - occupiedSlots, 0)
    return capacity
  }, {
    totalSlots: 0,
    occupiedSlots: 0,
    availableSlots: 0,
    unavailableSlots: 0,
  })
}
