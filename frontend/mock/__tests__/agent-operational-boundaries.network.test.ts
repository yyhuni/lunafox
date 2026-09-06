import { describe, expect, it } from "vitest"
import { agentService } from "@/services/agent.service"
import { getScan } from "@/services/scan.service"

const API_BASE = "http://localhost/v1"

describe("Agent operational mock network boundaries", () => {
  it("intercepts every authoritative Agent boundary with backend-shaped payloads", async () => {
    const [summaryResponse, mapResponse, createTokenResponse, tokenResponse, agentResponse, scanResponse] =
      await Promise.all([
        fetch(`${API_BASE}/admin/agentClusterSummaries/current`),
        fetch(`${API_BASE}/admin/agentLocationMaps/current`),
        fetch(`${API_BASE}/admin/agentRegistrationTokens`, { method: "POST" }),
        fetch(`${API_BASE}/admin/agentRegistrationTokens/101`),
        fetch(`${API_BASE}/admin/agents/1`),
        fetch(`${API_BASE}/scans/1`),
      ])

    expect(summaryResponse.status).toBe(200)
    expect(mapResponse.status).toBe(200)
    expect(createTokenResponse.status).toBe(201)
    expect(tokenResponse.status).toBe(200)
    expect(agentResponse.status).toBe(200)
    expect(scanResponse.status).toBe(200)

    const [summary, locationMap, createdToken, token, agent, scan] = await Promise.all([
      summaryResponse.json(),
      mapResponse.json(),
      createTokenResponse.json(),
      tokenResponse.json(),
      agentResponse.json(),
      scanResponse.json(),
    ]) as Array<Record<string, unknown>>

    expect(summary).toMatchObject({
      name: "agentClusterSummaries/current",
      totalNodes: 10,
      locationCoverage: { positionedCount: 8, unpositionedCount: 2 },
    })
    expect(locationMap).toMatchObject({
      name: "agentLocationMaps/current",
      serverLocation: { state: "current", providerKey: "freeipapi", accuracyRadiusKm: null },
    })
    expect(locationMap.agents).toHaveLength(8)
    expect(createdToken).toMatchObject({
      name: "agentRegistrationTokens/101",
      token: "a1b2c3d4",
    })
    expect(token).toMatchObject({
      name: "agentRegistrationTokens/101",
      state: "active",
    })
    expect(token).not.toHaveProperty("token")
    expect(agent).toMatchObject({
      name: "agents/1",
      observedSourceIp: "1.1.1.1",
      observedIpGeneration: 1,
      locationState: "current",
    })
    expect(scan).toMatchObject({
      agent: "agents/101",
      agentName: "scan-agent-east-01.example.internal",
      agentStatus: "online",
      agentHealthState: "healthy",
      agentDeleted: false,
      assignmentMode: "automatic",
    })
  })

  it("does not retain the removed nested registration-token route", async () => {
    const response = await fetch(`${API_BASE}/admin/agents/registrationTokens`, { method: "POST" })
    const payload = await response.json() as { error?: { code?: string } }

    expect(response.status).toBe(501)
    expect(payload.error?.code).toBe("mock_route_not_implemented")
  })

  it("round-trips mock payloads through the production service adapters", async () => {
    const createdToken = await agentService.createRegistrationToken()
    const [summary, locationMap, token, agent, scan] = await Promise.all([
      agentService.getAgentClusterSummary(),
      agentService.getAgentLocationMap(),
      agentService.getRegistrationToken(createdToken.resourceName),
      agentService.getAgent("agents/1"),
      getScan(1),
    ])

    expect(summary).toMatchObject({
      resourceName: "agentClusterSummaries/current",
      totalNodes: 10,
    })
    expect(locationMap.resourceName).toBe("agentLocationMaps/current")
    expect(locationMap.agents).toHaveLength(8)
    expect(locationMap.agents[0]).toMatchObject({
      resourceName: "agents/1",
      location: { providerKey: "freeipapi" },
    })
    expect(token).toMatchObject({
      resourceName: "agentRegistrationTokens/101",
      agents: [{ resourceName: "agents/1" }, { resourceName: "agents/2" }],
    })
    expect(token).not.toHaveProperty("token")
    expect(agent).toMatchObject({ resourceName: "agents/1", locationState: "current" })
    expect(scan).toMatchObject({ agent: "agents/101", agentDeleted: false })
  })
})
