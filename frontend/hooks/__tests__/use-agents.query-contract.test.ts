import { act, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  agentKeys,
  useAgent,
  useAgentClusterSummary,
  useAgentFilterOptions,
  useAgentLocationMap,
  useAgents,
  useRegistrationToken,
  useSelectedAgentDetail,
} from "@/hooks/use-agents"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"

const agentServiceMocks = vi.hoisted(() => ({
  getAgents: vi.fn(),
  getAgentFilterOptions: vi.fn(),
  getAgent: vi.fn(),
  getAgentClusterSummary: vi.fn(),
  getAgentLocationMap: vi.fn(),
  getRegistrationToken: vi.fn(),
  deleteAgent: vi.fn(),
  updateDistributedConfig: vi.fn(),
  createRegistrationToken: vi.fn(),
}))

vi.mock("@/services/agent.service", () => ({
  agentService: agentServiceMocks,
}))

describe("use-agents query contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("agent settings list 查询走 canonical collection 契约", async () => {
    agentServiceMocks.getAgents.mockResolvedValue({
      results: [
        {
          id: 1,
          name: "agent-node-1",
          status: "online",
          maxTasks: 3,
          cpuThreshold: 80,
          memThreshold: 80,
          diskThreshold: 90,
          health: { state: "ok" },
          createdAt: "2026-03-06T00:00:00Z",
        },
      ],
      totalSize: 1,
      pageSize: 100,
    })

    const params = {
      pageSize: 100,
      pageToken: "token-page-2",
      filter: `(displayName="edge" || observedHostname="edge" || connectionIp="edge")`,
      orderBy: "createdAt desc",
    }
    const { result } = renderHookWithProviders(() => useAgents(params))

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(agentServiceMocks.getAgents).toHaveBeenCalledWith(params)
    expect(result.current.data?.results[0]).toMatchObject({
      id: 1,
      name: "agent-node-1",
      status: "online",
    })
  })

  it("disabled collection query 不请求节点或启动轮询", () => {
    renderHookWithProviders(() => useAgents({ pageSize: 100 }, { enabled: false }))

    expect(agentServiceMocks.getAgents).not.toHaveBeenCalled()
  })

  it("agent filter options 使用独立后端 options 契约", async () => {
    agentServiceMocks.getAgentFilterOptions.mockResolvedValue({
      results: [{ value: "healthy", label: "healthy", count: 8 }],
    })

    const { result } = renderHookWithProviders(() => useAgentFilterOptions("healthState"))

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(agentServiceMocks.getAgentFilterOptions).toHaveBeenCalledWith("healthState")
    expect(result.current.data?.results[0]?.value).toBe("healthy")
  })

  it("agent node 详情查询按 id 读取受支持主线接口", async () => {
    agentServiceMocks.getAgent.mockResolvedValue({
      id: 7,
      name: "agent-node-7",
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

    expect(agentServiceMocks.getAgent).toHaveBeenCalledWith("agents/7", expect.any(AbortSignal))
    expect(result.current.data?.id).toBe(7)
  })

  it("shares fixed summary/map keys and distinguishes initial query failure", async () => {
    agentServiceMocks.getAgentLocationMap.mockRejectedValue(new Error("map unavailable"))

    const { result } = renderHookWithProviders(() => useAgentLocationMap())

    await waitFor(() => {
      expect(result.current.isInitialError).toBe(true)
    })
    expect(result.current.isRefetchStale).toBe(false)
    expect(result.current.lastSuccessfulAt).toBeNull()
    expect(agentKeys.clusterSummary()).toEqual(["agentNodes", "clusterSummary"])
    expect(agentKeys.locationMap()).toEqual(["agentNodes", "locationMap"])
  })

  it("retains the last successful summary across refetch failure and clears stale metadata on recovery", async () => {
    const initial = { resourceName: "agentClusterSummaries/current", totalNodes: 10 }
    agentServiceMocks.getAgentClusterSummary.mockResolvedValueOnce(initial)
    const { result } = renderHookWithProviders(() => useAgentClusterSummary())

    await waitFor(() => {
      expect(result.current.data).toBe(initial)
    })
    const firstSuccessfulAt = result.current.lastSuccessfulAt
    expect(firstSuccessfulAt).toEqual(expect.any(Number))

    agentServiceMocks.getAgentClusterSummary.mockRejectedValueOnce(new Error("refresh failed"))
    await act(async () => {
      await result.current.refetch()
    })
    await waitFor(() => {
      expect(result.current.isRefetchStale).toBe(true)
    })
    expect(result.current.isInitialError).toBe(false)
    expect(result.current.data).toBe(initial)
    expect(result.current.lastSuccessfulAt).toBe(firstSuccessfulAt)

    const recovered = { resourceName: "agentClusterSummaries/current", totalNodes: 11 }
    agentServiceMocks.getAgentClusterSummary.mockResolvedValueOnce(recovered)
    await act(async () => {
      await result.current.refetch()
    })
    await waitFor(() => {
      expect(result.current.data).toEqual(recovered)
      expect(result.current.data?.totalNodes).toBe(11)
      expect(result.current.isRefetchStale).toBe(false)
    })
  })

  it("keys registration-token status only by its non-secret resource identity", async () => {
    const registrationToken = {
      resourceName: "agentRegistrationTokens/101",
      token: "bearer-secret-must-not-enter-cache",
    }
    agentServiceMocks.getRegistrationToken.mockResolvedValue({
      id: 101,
      resourceName: registrationToken.resourceName,
      expiresAt: "2099-08-04T08:00:00Z",
      state: "active",
      agents: [],
    })

    const { queryClient, result } = renderHookWithProviders(() => useRegistrationToken(registrationToken))

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
    expect(agentServiceMocks.getRegistrationToken).toHaveBeenCalledWith(
      registrationToken.resourceName,
      expect.any(AbortSignal),
    )
    const serializedKeys = JSON.stringify(queryClient.getQueryCache().getAll().map((query) => query.queryKey))
    expect(serializedKeys).toContain(registrationToken.resourceName)
    expect(serializedKeys).not.toContain(registrationToken.token)
  })

  it("restores a selected Agent through the canonical detail key while collections stay untouched", async () => {
    agentServiceMocks.getAgent.mockResolvedValue({ id: 1001, resourceName: "agents/1001" })

    const { result } = renderHookWithProviders(() => useSelectedAgentDetail("agents/1001"))

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
    expect(agentServiceMocks.getAgent).toHaveBeenCalledWith("agents/1001", expect.any(AbortSignal))
    expect(agentServiceMocks.getAgents).not.toHaveBeenCalled()
    expect(agentKeys.detail("agents/1001")).toEqual(["agentNodes", "detail", "agents/1001"])
  })
})
