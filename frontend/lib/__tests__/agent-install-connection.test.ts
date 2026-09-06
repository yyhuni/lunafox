import { describe, expect, it } from "vitest"
import {
  projectAgentInstallConnection,
} from "@/lib/agent-install-connection"
import type { Agent } from "@/types/agent.types"

function createAgent(overrides: Partial<Agent> & Pick<Agent, "id">): Agent {
  return {
    name: `agents/${overrides.id}`,
    displayName: `agent-${overrides.id}`,
    status: "offline",
    maxTasks: 3,
    cpuThreshold: 80,
    memThreshold: 80,
    diskThreshold: 80,
    health: { state: "unknown" },
    createdAt: `2026-07-27T00:00:${String(overrides.id).padStart(2, "0")}Z`,
    ...overrides,
    id: overrides.id,
  }
}

describe("agent installation connection state", () => {
  it("uses the complete token-scoped projection without requiring a heartbeat compatibility field", () => {
    const state = projectAgentInstallConnection([
      createAgent({ id: 3, status: "online", lastHeartbeat: null }),
    ])

    expect(state.phase).toBe("online")
    expect(state.agents[0]?.status).toBe("online")
  })

  it("keeps a mixed token-scoped group registered until every attributed Agent is online", () => {
    const state = projectAgentInstallConnection([
      createAgent({ id: 3, status: "online", lastHeartbeat: "2026-07-27T00:01:00Z" }),
      createAgent({ id: 4, status: "offline" }),
    ])

    expect(state.phase).toBe("registered")
    expect(state.agents.map((agent) => agent.status)).toEqual(["online", "registered"])
  })

  it("replaces membership with the latest complete projection and keeps deterministic order", () => {
    const state = projectAgentInstallConnection([
      createAgent({ id: 4, status: "online", lastHeartbeat: "2026-07-27T00:01:00Z" }),
      createAgent({
        id: 3,
        connectionIp: "10.0.0.3",
        status: "online",
        heartbeat: {
          cpu: 1,
          mem: 2,
          disk: 3,
          runningTasks: 0,
          taskSlotsUsed: 0,
          uptime: 10,
          updatedAt: "2026-07-27T00:01:00Z",
        },
      }),
    ])

    expect(state.phase).toBe("online")
    expect(state.agents.map((agent) => [agent.id, agent.status])).toEqual([
      [3, "online"],
      [4, "online"],
    ])
    expect(state.agents.find((agent) => agent.id === 3)?.address).toBe("10.0.0.3")
  })

  it("does not substitute an observed hostname for a missing connection address", () => {
    const state = projectAgentInstallConnection([
      createAgent({ id: 3, observedHostname: "docker-agent", status: "online" }),
    ])

    expect(state.agents[0]?.address).toBe("")
  })

  it("treats an authoritative empty projection as waiting", () => {
    expect(projectAgentInstallConnection([])).toEqual({ phase: "waiting", agents: [] })
  })
})
