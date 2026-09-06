import { describe, expect, it } from "vitest"

import {
  buildAgentLocationMapNodes,
  isValidAgentLocationCoordinate,
} from "@/components/overview/agent-location-map-data"
import type { AgentLocationMapAgent } from "@/types/agent.types"

function makeAgent(id: number, overrides: Partial<AgentLocationMapAgent> = {}): AgentLocationMapAgent {
  return {
    id,
    resourceName: `agents/${id}`,
    displayName: `agent-${id}`,
    status: "online",
    healthState: "healthy",
    taskSlotsUsed: 0,
    location: {
      state: "current",
      latitude: 39.9042,
      longitude: 116.4074,
      accuracyRadiusKm: null,
      sourceObservedIp: "1.1.1.1",
      providerKey: "freeipapi",
      resolvedAt: "2026-08-02T07:00:00Z",
    },
    ...overrides,
  }
}

describe("agent-location-map-data", () => {
  it("rejects missing, non-finite, and out-of-range coordinates", () => {
    expect(isValidAgentLocationCoordinate(undefined, 0)).toBe(false)
    expect(isValidAgentLocationCoordinate(Number.NaN, 0)).toBe(false)
    expect(isValidAgentLocationCoordinate(91, 0)).toBe(false)
    expect(isValidAgentLocationCoordinate(0, -181)).toBe(false)
    expect(isValidAgentLocationCoordinate(39.9, 116.4)).toBe(true)
  })

  it("maps complete backend projections beyond a collection page and preserves freshness/provenance", () => {
    const nodes = buildAgentLocationMapNodes([
      makeAgent(1),
      makeAgent(1001, {
        healthState: "warning",
        taskSlotsUsed: 3,
        location: {
          ...makeAgent(1001).location,
          state: "expired",
          latitude: 35.6762,
          longitude: 139.6503,
          sourceObservedIp: "8.8.8.8",
        },
      }),
      makeAgent(1002, { status: "offline", healthState: "critical", taskSlotsUsed: null }),
    ])

    expect(nodes).toHaveLength(3)
    expect(nodes.map((node) => node.status)).toEqual(["healthy", "warning", "offline"])
    expect(nodes[1]).toMatchObject({
      id: "agent-location-1001",
      resourceName: "agents/1001",
      locationState: "expired",
      sourceObservedIp: "8.8.8.8",
      taskSlotsUsed: 3,
      hasTaskLoadData: true,
    })
    expect(nodes[2]?.hasTaskLoadData).toBe(false)
  })

  it("drops invalid points instead of manufacturing a default coordinate", () => {
    const invalid = makeAgent(7, {
      location: {
        ...makeAgent(7).location,
        latitude: Number.NaN,
      },
    })

    expect(buildAgentLocationMapNodes([invalid])).toEqual([])
  })
})
