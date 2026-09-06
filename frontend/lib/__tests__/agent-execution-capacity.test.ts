import { describe, expect, it } from "vitest"
import { buildAgentExecutionCapacity } from "@/lib/agent-execution-capacity"
import type { Agent } from "@/types/agent.types"

function makeAgent(overrides: Partial<Agent> = {}): Agent {
  return {
    id: 1,
    name: "agent-01",
    status: "online",
    maxTasks: 4,
    cpuThreshold: 80,
    memThreshold: 80,
    diskThreshold: 85,
    health: { state: "healthy" },
    heartbeat: {
      cpu: 10,
      mem: 10,
      disk: 10,
      runningTasks: 1,
      taskSlotsUsed: 2,
      uptime: 60,
      updatedAt: "2026-07-27T00:00:00Z",
    },
    createdAt: "2026-07-27T00:00:00Z",
    ...overrides,
  }
}

describe("agent execution capacity", () => {
  it("uses task slot usage rather than running task count", () => {
    expect(buildAgentExecutionCapacity([
      makeAgent({ maxTasks: 5, heartbeat: { ...makeAgent().heartbeat!, runningTasks: 2, taskSlotsUsed: 3 } }),
      makeAgent({ id: 2, maxTasks: 4, heartbeat: { ...makeAgent().heartbeat!, runningTasks: 1, taskSlotsUsed: 1 } }),
    ])).toEqual({
      totalSlots: 9,
      occupiedSlots: 4,
      availableSlots: 5,
      unavailableSlots: 0,
    })
  })

  it("marks offline and heartbeat-less Agent slots unavailable", () => {
    expect(buildAgentExecutionCapacity([
      makeAgent({ maxTasks: 4, status: "offline" }),
      makeAgent({ id: 2, maxTasks: 3, heartbeat: undefined }),
      makeAgent({ id: 3, maxTasks: 2, heartbeat: { ...makeAgent().heartbeat!, taskSlotsUsed: 1 } }),
    ])).toEqual({
      totalSlots: 9,
      occupiedSlots: 1,
      availableSlots: 1,
      unavailableSlots: 7,
    })
  })
})
