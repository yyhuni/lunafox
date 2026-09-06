import { beforeEach, describe, expect, it, vi } from "vitest"

const apiMocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock("@/lib/api-client", () => ({
  api: apiMocks,
}))

import { api } from "@/lib/api-client"
import { agentService } from "@/services/agent.service"

function makeAgentDto(id = 1) {
  return {
    id,
    name: `agents/${id}`,
    displayName: `edge-${id}`,
    status: "online",
    connectionIp: "172.20.0.5",
    maxTasks: 3,
    cpuThreshold: 80,
    memThreshold: 80,
    diskThreshold: 90,
    health: { state: "healthy" },
    createdAt: "2026-06-19T00:00:00Z",
  }
}

describe("agent.service contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("uses canonical collection params and normalizes agent list responses", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [
          {
            id: 1,
            name: "agents/1",
            displayName: "edge-01",
            status: "online",
            connectionIp: "172.20.0.5",
            maxTasks: 3,
            cpuThreshold: 80,
            memThreshold: 80,
            diskThreshold: 90,
            health: { state: "ok" },
            createdAt: "2026-06-19T00:00:00Z",
          },
        ],
        totalSize: 137,
        nextPageToken: "next-agent-page",
      },
    } as never)

    const result = await agentService.getAgents({
      pageSize: 25,
      pageToken: "token-page-2",
      filter: `(displayName="edge" || observedHostname="edge" || connectionIp="edge") && status=="online"`,
      orderBy: "createdAt desc",
    })

    expect(api.get).toHaveBeenCalledWith("/admin/agents", {
      params: {
        pageSize: 25,
        pageToken: "token-page-2",
        filter: `(displayName="edge" || observedHostname="edge" || connectionIp="edge") && status=="online"`,
        orderBy: "createdAt desc",
      },
    })
    expect(result.totalSize).toBe(137)
    expect(result.total).toBe(137)
    expect(result.page).toBe(1)
    expect(result.pageSize).toBe(25)
    expect(result.nextPageToken).toBe("next-agent-page")
    expect(result.results[0]).toMatchObject({
      id: 1,
      name: "edge-01",
      displayName: "edge-01",
      resourceName: "agents/1",
      connectionIp: "172.20.0.5",
    })
    expect(result.results[0]).not.toHaveProperty("ipAddress")
  })

  it("forwards an AbortSignal for session-scoped Agent list queries", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: { results: [], totalSize: 0, pageSize: 50 },
    } as never)
    const controller = new AbortController()

    await agentService.getAgents({ pageSize: 50, orderBy: "createdAt desc" }, controller.signal)

    expect(api.get).toHaveBeenCalledWith("/admin/agents", {
      params: { pageSize: 50, orderBy: "createdAt desc" },
      signal: controller.signal,
    })
  })

  it("preserves an empty raw health state before the first Agent heartbeat", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [{
          ...makeAgentDto(2),
          status: "offline",
          health: { state: "" },
        }],
        totalSize: 1,
      },
    } as never)

    const result = await agentService.getAgents({ pageSize: 50, orderBy: "createdAt desc" })

    expect(result.results[0]).toMatchObject({
      id: 2,
      status: "offline",
      health: { state: "" },
    })
  })

  it("fetches backend-sourced agent filter options", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [
          { value: "online", label: "online", count: 8 },
          { value: "offline", label: "offline", count: 2 },
        ],
      },
    } as never)

    const result = await agentService.getAgentFilterOptions("status")

    expect(api.get).toHaveBeenCalledWith("/admin/agents/filterOptions", {
      params: { field: "status" },
    })
    expect(result.results).toHaveLength(2)
  })

  it("adapts canonical Agent detail identity and location provenance", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        ...makeAgentDto(7),
        observedSourceIp: "1.1.1.1",
        observedIpGeneration: 4,
        locationState: "current",
        location: {
          latitude: 39.904212345,
          longitude: 116.407498765,
          accuracyRadiusKm: null,
          sourceObservedIp: "1.1.1.1",
          providerKey: "freeipapi",
          resolvedAt: "2026-08-01T00:00:00Z",
        },
      },
    } as never)
    const controller = new AbortController()

    const result = await agentService.getAgent("agents/7", controller.signal)

    expect(api.get).toHaveBeenCalledWith("/admin/agents/7", { signal: controller.signal })
    expect(result).toMatchObject({
      id: 7,
      name: "edge-7",
      resourceName: "agents/7",
      observedSourceIp: "1.1.1.1",
      observedIpGeneration: 4,
      locationState: "current",
      location: {
        latitude: 39.904212345,
        accuracyRadiusKm: null,
        providerKey: "freeipapi",
      },
    })
  })

  it("reads and validates the complete fixed cluster summary", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        name: "agentClusterSummaries/current",
        generatedAt: "2026-08-04T00:00:00Z",
        executionFreshnessSeconds: 15,
        totalNodes: 10,
        healthyCount: 6,
        warningCount: 2,
        offlineCount: 2,
        unknownCount: 0,
        staleAgentCount: 1,
        executionCapacity: {
          configuredSlots: 70,
          occupiedSlots: 21,
          availableSlots: 27,
          unavailableSlots: 22,
          overcommittedSlots: 0,
        },
        clusterState: "needsAttention",
        reasonCodes: ["offline_agents", "warning_agents", "stale_runtime_observations"],
        locationCoverage: { positionedCount: 8, unpositionedCount: 2 },
      },
    } as never)

    const result = await agentService.getAgentClusterSummary()

    expect(api.get).toHaveBeenCalledWith("/admin/agentClusterSummaries/current", undefined)
    expect(result.resourceName).toBe("agentClusterSummaries/current")
    expect(result.executionCapacity).toEqual({
      configuredSlots: 70,
      occupiedSlots: 21,
      availableSlots: 27,
      unavailableSlots: 22,
      overcommittedSlots: 0,
    })
  })

  it("adapts a complete location map and preserves nullable radius", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        name: "agentLocationMaps/current",
        generatedAt: "2026-08-04T00:00:00Z",
        serverLocation: null,
        agents: [{
          name: "agents/7",
          displayName: "edge-7",
          status: "offline",
          healthState: "healthy",
          taskSlotsUsed: null,
          location: {
            state: "expired",
            latitude: 1.23456789,
            longitude: 103.98765432,
            accuracyRadiusKm: null,
            sourceObservedIp: "1.1.1.1",
            providerKey: "freeipapi",
            resolvedAt: "2026-07-01T00:00:00Z",
          },
        }],
      },
    } as never)

    const result = await agentService.getAgentLocationMap()

    expect(api.get).toHaveBeenCalledWith("/admin/agentLocationMaps/current", undefined)
    expect(result).toMatchObject({
      resourceName: "agentLocationMaps/current",
      serverLocation: null,
      agents: [{
        id: 7,
        resourceName: "agents/7",
        taskSlotsUsed: null,
        location: { state: "expired", latitude: 1.23456789, accuracyRadiusKm: null },
      }],
    })
  })

  it("creates and reads registration-token resources through canonical non-secret paths", async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: {
        name: "agentRegistrationTokens/101",
        token: "one-time-secret",
        expiresAt: "2099-08-04T08:00:00Z",
      },
    } as never)
    const created = await agentService.createRegistrationToken()

    expect(api.post).toHaveBeenCalledWith("/admin/agentRegistrationTokens")
    expect(created).toEqual({
      id: 101,
      resourceName: "agentRegistrationTokens/101",
      token: "one-time-secret",
      expiresAt: "2099-08-04T08:00:00Z",
    })

    vi.mocked(api.get).mockResolvedValue({
      data: {
        name: "agentRegistrationTokens/101",
        expiresAt: "2099-08-04T08:00:00Z",
        state: "active",
        agents: [makeAgentDto(7)],
      },
    } as never)
    const resource = await agentService.getRegistrationToken(created.resourceName)

    expect(api.get).toHaveBeenCalledWith("/admin/agentRegistrationTokens/101", undefined)
    expect(resource).toMatchObject({
      id: 101,
      resourceName: "agentRegistrationTokens/101",
      state: "active",
      agents: [{ id: 7, resourceName: "agents/7", name: "edge-7" }],
    })
    expect(resource).not.toHaveProperty("token")
  })

  it("fails closed on malformed canonical names, coordinates, and leaked status secrets", async () => {
    await expect(agentService.getRegistrationToken("one-time-secret")).rejects.toThrow(
      "Agent API response is invalid at registrationToken.name",
    )
    expect(api.get).not.toHaveBeenCalled()

    vi.mocked(api.get).mockResolvedValueOnce({
      data: {
        results: [{ ...makeAgentDto(1), ipAddress: "10.0.0.1" }],
        totalSize: 1,
      },
    } as never)
    await expect(agentService.getAgents()).rejects.toThrow(
      "Agent API response is invalid at results[0].ipAddress",
    )

    vi.mocked(api.get).mockResolvedValueOnce({
      data: {
        name: "agentLocationMaps/current",
        generatedAt: "2026-08-04T00:00:00Z",
        serverLocation: null,
        agents: [{
          name: "agents/1",
          displayName: "edge-1",
          status: "online",
          healthState: "healthy",
          taskSlotsUsed: 0,
          location: {
            state: "current",
            latitude: 91,
            longitude: 0,
            accuracyRadiusKm: null,
            sourceObservedIp: "1.1.1.1",
            providerKey: "freeipapi",
            resolvedAt: "2026-08-01T00:00:00Z",
          },
        }],
      },
    } as never)
    await expect(agentService.getAgentLocationMap()).rejects.toThrow(
      "Agent API response is invalid at agentLocationMap.agents[0].location.latitude",
    )

    vi.mocked(api.get).mockResolvedValueOnce({
      data: {
        name: "agentRegistrationTokens/101",
        token: "must-not-return",
        expiresAt: "2099-08-04T08:00:00Z",
        state: "active",
        agents: [],
      },
    } as never)
    await expect(agentService.getRegistrationToken("agentRegistrationTokens/101")).rejects.toThrow(
      "Agent API response is invalid at registrationToken.token",
    )
  })
})
