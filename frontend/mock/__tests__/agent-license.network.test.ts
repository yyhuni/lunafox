import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { agentService } from "@/services/agent.service"
import { activateAgentLicense } from "@/services/agent-license.service"
import { getMockScenario, setMockScenario } from "@/mock/scenarios"

const endpoint = "http://localhost/v1/admin/agentLicenses/current:activate"

describe("Agent license mock network", () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ["Date"] })
    vi.setSystemTime(new Date("2026-10-19T00:00:00.000Z"))
  })
  afterEach(() => vi.useRealTimers())
  it.each([1, 2, 3])("activates a %i-month Key and refreshes the authoritative summary without changing nodes or occupied work", async (months) => {
    const before = await agentService.getAgentClusterSummary()
    const activated = await activateAgentLicense(`  LF-DEMO-${months}M  `)
    const after = await agentService.getAgentClusterSummary()
    expect(before.agentLimit).toBe(3)
    expect(activated).toMatchObject({ resourceName: "agentLicenses/current", state: "active", months, agentLimit: 10 })
    expect(after.license).toEqual(activated)
    expect(after.agentLimit).toBe(10)
    expect(after.totalNodes).toBe(before.totalNodes)
    expect(after.executionCapacity.configuredSlots).toBe(before.executionCapacity.configuredSlots)
    expect(after.executionCapacity.occupiedSlots).toBe(before.executionCapacity.occupiedSlots)
    expect(after.executionCapacity.availableSlots).toBeGreaterThan(before.executionCapacity.availableSlots)
    const replay = await activateAgentLicense(`LF-DEMO-${months}M`)
    expect(replay).toEqual(activated)
  })

  it("preserves an active license when an expired, future or invalid Key is submitted", async () => {
    const active = await activateAgentLicense("LF-DEMO-2M")
    for (const [key, reason] of [["wrong", "LICENSE_KEY_INVALID"], ["LF-DEMO-EXPIRED", "LICENSE_KEY_EXPIRED"], ["LF-DEMO-NOT-YET-VALID", "LICENSE_KEY_NOT_YET_VALID"]]) {
      const response = await fetch(endpoint, { method: "POST", body: JSON.stringify({ key }) })
      expect(response.status).toBe(400)
      expect(await response.json()).toMatchObject({ error: { code: 400, status: "INVALID_ARGUMENT", details: [{ reason }] } })
      expect((await agentService.getAgentClusterSummary()).license).toEqual(active)
    }
  })

  it("provides expired/stress scenarios and renews an expired license", async () => {
    setMockScenario("edge")
    const expired = await agentService.getAgentClusterSummary()
    expect(expired).toMatchObject({ agentLimit: 3, totalNodes: 10, license: { state: "expired" } })
    await activateAgentLicense("LF-DEMO-3M")
    expect((await agentService.getAgentClusterSummary()).license?.state).toBe("active")
    setMockScenario("stress")
    expect((await agentService.getAgentClusterSummary()).license?.state).toBe("active")
    expect(getMockScenario()).toBe("stress")
  })

  it("rejects missing/extra payload fields instead of pretending activation succeeded", async () => {
    for (const body of [{}, { key: "" }, { key: "LF-DEMO-1M", agentLimit: 999 }]) {
      const response = await fetch(endpoint, { method: "POST", body: JSON.stringify(body) })
      expect(response.status).toBe(400)
    }
    expect((await agentService.getAgentClusterSummary()).agentLimit).toBe(3)
  })

  it("returns an explicit network failure in the error scenario", async () => {
    setMockScenario("error")
    const response = await fetch(endpoint, { method: "POST", body: JSON.stringify({ key: "LF-DEMO-1M" }) })
    expect(response.status).toBe(503)
  })
  it.each(["scans", "scans:batchCreate", "scans:quickCreate"])("rejects restricted pinned work at %s with the live ErrorInfo contract", async (path) => {
    const response = await fetch(`http://localhost/v1/${path}`, { method: "POST", body: JSON.stringify({ agent: "agents/4" }) })
    expect(response.status).toBe(400)
    expect(await response.json()).toMatchObject({ error: { code: 400, status: "FAILED_PRECONDITION", details: [{ "@type": "type.googleapis.com/google.rpc.ErrorInfo", domain: "lunafox", reason: "AGENT_LICENSE_RESTRICTED" }] } })
  })

})
