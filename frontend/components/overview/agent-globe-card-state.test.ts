import { describe, expect, it } from "vitest"
import type { Agent } from "@/types/agent.types"
import { buildAgentSummary } from "./agent-globe-card-state"

function makeNode(overrides: Partial<Agent>): Agent {
  return {
    id: 1,
    name: "agent-edge-01",
    status: "online",
    maxTasks: 5,
    cpuThreshold: 85,
    memThreshold: 85,
    diskThreshold: 90,
    health: { state: "ok" },
    createdAt: "2024-12-01T00:00:00Z",
    ...overrides,
  }
}

describe("buildAgentSummary", () => {
  it("summarizes total, alive, high load and offline nodes from current fields", () => {
    const summary = buildAgentSummary([
      makeNode({ id: 1, status: "online", health: { state: "ok" } }),
      makeNode({ id: 2, status: "online", health: { state: "warning" } }),
      makeNode({ id: 3, status: "offline", health: { state: "error" } }),
      makeNode({
        id: 4,
        status: "online",
        health: { state: "ok" },
        heartbeat: { cpu: 91, mem: 40, disk: 30, runningTasks: 2, taskSlotsUsed: 2, uptime: 120, updatedAt: "2024-12-01T00:00:00Z" },
        cpuThreshold: 90,
      }),
    ])

    expect(summary).toEqual({
      total: 4,
      alive: 3,
      highLoad: 2,
      offline: 1,
    })
  })
})
