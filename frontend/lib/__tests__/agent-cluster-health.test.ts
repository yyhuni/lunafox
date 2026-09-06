import { describe, expect, it } from "vitest"
import {
  buildAgentClusterHealthSummary,
  type AgentClusterHealthStats,
} from "@/lib/agent-cluster-health"
import type { AgentExecutionCapacity } from "@/lib/agent-execution-capacity"

const healthyStats: AgentClusterHealthStats = {
  total: 4,
  online: 4,
  offline: 0,
  warning: 0,
  healthy: 4,
  unknown: 0,
}

function makeCapacity(overrides: Partial<AgentExecutionCapacity> = {}): AgentExecutionCapacity {
  return {
    totalSlots: 10,
    occupiedSlots: 4,
    availableSlots: 6,
    unavailableSlots: 0,
    ...overrides,
  }
}

function buildSummary({
  stats = healthyStats,
  capacity = makeCapacity(),
  isComplete = true,
}: {
  stats?: AgentClusterHealthStats
  capacity?: AgentExecutionCapacity
  isComplete?: boolean
} = {}) {
  return buildAgentClusterHealthSummary({ stats, capacity, isComplete })
}

describe("agent cluster health", () => {
  it("reports an empty or incomplete overview as unknown", () => {
    expect(buildSummary({
      stats: { ...healthyStats, total: 0, online: 0, healthy: 0 },
    })).toMatchObject({ state: "unknown", isComplete: true })

    expect(buildSummary({ isComplete: false })).toMatchObject({
      state: "unknown",
      isComplete: false,
    })
  })

  it("reports no online Agents or no available configured slots as critical", () => {
    expect(buildSummary({
      stats: { ...healthyStats, online: 0, offline: 4, healthy: 0 },
    })).toMatchObject({ state: "critical", attentionCount: 4 })

    expect(buildSummary({
      capacity: makeCapacity({ availableSlots: 0 }),
    })).toMatchObject({ state: "critical" })
  })

  it("reports offline or warning Agents as needing attention", () => {
    expect(buildSummary({
      stats: { ...healthyStats, online: 3, offline: 1, healthy: 3 },
    })).toMatchObject({ state: "needsAttention", attentionCount: 1 })

    expect(buildSummary({
      stats: { ...healthyStats, warning: 1, healthy: 3 },
    })).toMatchObject({ state: "needsAttention", attentionCount: 1 })
  })

  it("treats the 20 percent available-capacity boundary as needing attention", () => {
    expect(buildSummary({
      capacity: makeCapacity({ totalSlots: 10, occupiedSlots: 8, availableSlots: 2 }),
    })).toMatchObject({ state: "needsAttention" })
  })

  it("reports a complete unconstrained cluster as healthy", () => {
    expect(buildSummary()).toEqual({
      state: "healthy",
      attentionCount: 0,
      isComplete: true,
    })
  })
})
