import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { activateMockAgentLicense, resetMockAgentLicense } from "../agent-license"
import { deleteMockAgent, getMockAgentById, getMockAgentClusterSummary, getMockAgents, mockAgents } from "../agents"
import { getMockScanById } from "../scans"
import { setMockScenario } from "@/mock/scenarios"

const initialAgents = structuredClone(mockAgents)
const eligibleIds = () => getMockAgents({ orderBy: "createdAt asc" }).results
  .filter((agent) => agent.executionEligibility === "eligible").map((agent) => agent.id)

describe("Agent execution eligibility projection", () => {
  beforeEach(() => {
    setMockScenario("happy")
    resetMockAgentLicense()
    vi.useFakeTimers()
    vi.setSystemTime(new Date("2026-10-20T00:00:00Z"))
  })
  afterEach(() => {
    mockAgents.splice(0, mockAgents.length, ...structuredClone(initialAgents))
    vi.useRealTimers()
    resetMockAgentLicense()
  })
  it("ranks all identities before filtering and paging, with offline identities occupying places", () => {
    expect(eligibleIds()).toEqual([3, 9, 1])
    expect(getMockAgentById(3)).toMatchObject({ status: "offline", executionEligibility: "eligible" })
    expect(getMockAgents({ pageSize: 1, filter: 'displayName="prod"' }).results[0]).toMatchObject({ id: 4, executionEligibility: "license_restricted" })
  })
  it("uses numeric ID to break a registration timestamp tie and promotes the next node after deletion", () => {
    mockAgents.find((agent) => agent.id === 8)!.createdAt = "2020-01-01T00:00:00Z"
    mockAgents.find((agent) => agent.id === 7)!.createdAt = "2020-01-01T00:00:00Z"
    expect(eligibleIds()).toEqual([7, 8, 3])
    deleteMockAgent(7)
    expect(eligibleIds()).toEqual([8, 3, 9])
    expect(getMockAgentClusterSummary().totalNodes).toBe(9)
  })
  it("recovers on activation, restricts at expiry, and resumes after renewal while preserving occupied work", () => {
    const base = getMockAgentClusterSummary().executionCapacity
    expect(getMockScanById(6)?.executionBlockedReason).toBe("agent_license_restricted")
    activateMockAgentLicense({ key: "LF-DEMO-1M" })
    expect(eligibleIds()).toHaveLength(10)
    expect(getMockScanById(6)?.executionBlockedReason).toBeNull()
    const expanded = getMockAgentClusterSummary().executionCapacity
    expect(expanded.occupiedSlots).toBe(base.occupiedSlots)
    expect(expanded.availableSlots).toBeGreaterThan(base.availableSlots)
    vi.setSystemTime(new Date("2026-11-19T00:00:00Z"))
    expect(eligibleIds()).toEqual([3, 9, 1])
    expect(getMockAgentClusterSummary().executionCapacity).toEqual(base)
    activateMockAgentLicense({ key: "LF-DEMO-2M" })
    expect(eligibleIds()).toHaveLength(10)
    expect(getMockAgentClusterSummary().executionCapacity.occupiedSlots).toBe(base.occupiedSlots)
  })
})
