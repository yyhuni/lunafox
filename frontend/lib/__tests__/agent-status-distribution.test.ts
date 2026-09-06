import { describe, expect, it } from "vitest"
import {
  buildAgentStatusDistribution,
  getAgentStatusDistributionStatus,
  getAgentStatusDistributionTone,
} from "@/lib/agent-status-distribution"
import type { Agent } from "@/types/agent.types"

function makeAgent(overrides: Partial<Agent>): Agent {
  return {
    id: 1,
    name: "agent-01",
    status: "online",
    maxTasks: 4,
    cpuThreshold: 80,
    memThreshold: 80,
    diskThreshold: 85,
    health: { state: "healthy" },
    createdAt: "2024-12-01T00:00:00Z",
    ...overrides,
  }
}

describe("agent status distribution", () => {
  it("uses the same mutually exclusive source buckets as node management", () => {
    const agents = [
      makeAgent({ id: 1, status: "online", health: { state: "healthy" } }),
      makeAgent({ id: 2, status: "online", health: { state: "warning" } }),
      makeAgent({ id: 3, status: "offline", health: { state: "healthy" } }),
      makeAgent({ id: 4, status: "maintenance", health: { state: "healthy" } }),
      makeAgent({ id: 5, status: "online", health: { state: "paused" } }),
    ]

    expect(buildAgentStatusDistribution(agents)).toEqual({
      healthy: 1,
      warning: 1,
      offline: 1,
      unknown: 2,
    })
  })

  it("gives offline transport precedence over warning health metadata", () => {
    expect(getAgentStatusDistributionStatus(makeAgent({
      status: "offline",
      health: { state: "warning" },
    }))).toBe("offline")
  })

  it("keeps nodes outside a paged summary explicit as unknown", () => {
    expect(buildAgentStatusDistribution([
      makeAgent({ id: 1, health: { state: "healthy" } }),
      makeAgent({ id: 2, health: { state: "warning" } }),
    ], 5)).toEqual({
      healthy: 1,
      warning: 1,
      offline: 0,
      unknown: 3,
    })
  })

  it("maps each source distribution status to the shared semantic tone", () => {
    expect(getAgentStatusDistributionTone("healthy")).toBe("success")
    expect(getAgentStatusDistributionTone("warning")).toBe("warning")
    expect(getAgentStatusDistributionTone("offline")).toBe("error")
    expect(getAgentStatusDistributionTone("unknown")).toBe("muted")
  })
})
