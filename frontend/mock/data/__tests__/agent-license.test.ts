import { afterEach, describe, expect, it, vi } from "vitest"
import { activateMockAgentLicense, getMockAgentLicense, resetMockAgentLicense } from "../agent-license"

describe("Agent license fixed expiry simulation", () => {
  afterEach(() => { vi.useRealTimers(); resetMockAgentLicense() })
  it("uses the issued expiry even after state reset and later import", () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date("2026-10-19T10:30:00.000Z"))
    const result = activateMockAgentLicense({ key: "LF-DEMO-1M" })
    expect(result).toMatchObject({ license: { expiresAt: "2026-11-19T00:00:00.000Z" } })
    resetMockAgentLicense()
    vi.setSystemTime(new Date("2026-11-01T10:30:00.000Z"))
    expect(activateMockAgentLicense({ key: "LF-DEMO-1M" })).toEqual(result)
  })
  it("restores base quota at expiry without allowing replay to extend the same Key", () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date("2026-10-19T00:00:00.000Z"))
    activateMockAgentLicense({ key: "LF-DEMO-1M" })
    vi.setSystemTime(new Date("2026-11-19T00:00:00.000Z"))
    expect(getMockAgentLicense()).toMatchObject({ state: "expired", agentLimit: 3 })
    expect(activateMockAgentLicense({ key: "LF-DEMO-1M" })).toEqual({ reason: "LICENSE_KEY_EXPIRED" })
    expect(activateMockAgentLicense({ key: "LF-DEMO-2M" })).toMatchObject({ license: { state: "active", agentLimit: 10 } })
  })
  it("rejects an expired Key after a reset rather than issuing a fresh term", () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date("2026-11-19T00:00:00.000Z"))
    expect(activateMockAgentLicense({ key: "LF-DEMO-1M" })).toEqual({ reason: "LICENSE_KEY_EXPIRED" })
    expect(getMockAgentLicense()).toMatchObject({ state: "inactive", agentLimit: 3 })
  })
})
