import { describe, expect, it } from "vitest"

import {
  buildOverviewAgentHealthMatrix,
  OVERVIEW_AGENT_HEALTH_MATRIX_MAX_TILES,
} from "@/lib/overview-agent-health-matrix"
import type { Agent } from "@/types/agent.types"

function makeAgent(id: number, status = "online", healthState = "healthy"): Agent {
  return {
    id,
    name: `agent-${id}`,
    status,
    maxTasks: 4,
    cpuThreshold: 80,
    memThreshold: 80,
    diskThreshold: 85,
    health: { state: healthState },
    createdAt: "2024-12-01T00:00:00Z",
  }
}

describe("overview Agent health matrix", () => {
  it("keeps one visible tile for each small-fleet Agent", () => {
    const matrix = buildOverviewAgentHealthMatrix([makeAgent(1)])

    expect(matrix).toEqual({
      nodes: [{ id: 1, label: "agent-1", status: "healthy" }],
      hiddenCount: 0,
      columnCount: 6,
    })
  })

  it("keeps all thirty node identities without converting them into ratio tiles", () => {
    const nodes = Array.from({ length: OVERVIEW_AGENT_HEALTH_MATRIX_MAX_TILES }, (_, index) => makeAgent(index + 1))
    const matrix = buildOverviewAgentHealthMatrix(nodes)

    expect(matrix.nodes).toHaveLength(OVERVIEW_AGENT_HEALTH_MATRIX_MAX_TILES)
    expect(matrix.hiddenCount).toBe(0)
    expect(matrix.columnCount).toBe(10)
  })

  it("adds columns for a compact ten-node fleet before it consumes a taller matrix", () => {
    const nodes = Array.from({ length: 10 }, (_, index) => makeAgent(index + 1))

    expect(buildOverviewAgentHealthMatrix(nodes).columnCount).toBe(7)
  })

  it("reserves the final tile for an overflow entry when the fleet exceeds the dashboard capacity", () => {
    const nodes = Array.from({ length: 31 }, (_, index) => makeAgent(index + 1))
    const matrix = buildOverviewAgentHealthMatrix(nodes, 31)

    expect(matrix.nodes).toHaveLength(OVERVIEW_AGENT_HEALTH_MATRIX_MAX_TILES - 1)
    expect(matrix.hiddenCount).toBe(2)
    expect(matrix.columnCount).toBe(10)
  })
})
