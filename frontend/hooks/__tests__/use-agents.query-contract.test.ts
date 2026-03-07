import { waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { useAgent, useAgents } from "@/hooks/use-agents"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"

const agentServiceMocks = vi.hoisted(() => ({
  getAgents: vi.fn(),
  getAgent: vi.fn(),
  deleteAgent: vi.fn(),
  updateAgentConfig: vi.fn(),
  createRegistrationToken: vi.fn(),
}))

vi.mock("@/services/agent.service", () => ({
  agentService: agentServiceMocks,
}))

describe("use-agents query contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("workers settings list 查询走 agent 契约", async () => {
    agentServiceMocks.getAgents.mockResolvedValue({
      results: [
        {
          id: 1,
          name: "agent-1",
          status: "online",
          maxTasks: 3,
          cpuThreshold: 80,
          memThreshold: 80,
          diskThreshold: 90,
          health: { state: "ok" },
          createdAt: "2026-03-06T00:00:00Z",
        },
      ],
      total: 1,
      page: 1,
      pageSize: 100,
      totalPages: 1,
    })

    const { result } = renderHookWithProviders(() => useAgents(1, 100))

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(agentServiceMocks.getAgents).toHaveBeenCalledWith(1, 100, undefined)
    expect(result.current.data?.results[0]).toMatchObject({
      id: 1,
      name: "agent-1",
      status: "online",
    })
  })

  it("agent 详情查询按 id 读取受支持主线接口", async () => {
    agentServiceMocks.getAgent.mockResolvedValue({
      id: 7,
      name: "agent-7",
      status: "offline",
      maxTasks: 3,
      cpuThreshold: 80,
      memThreshold: 80,
      diskThreshold: 90,
      health: { state: "offline" },
      createdAt: "2026-03-06T00:00:00Z",
    })

    const { result } = renderHookWithProviders(() => useAgent(7))

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(agentServiceMocks.getAgent).toHaveBeenCalledWith(7)
    expect(result.current.data?.id).toBe(7)
  })
})
