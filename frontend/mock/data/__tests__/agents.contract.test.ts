import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "mock/data/agents.ts"), "utf8")

describe("agents contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function getMockAgents")
    expect(source).toContain("export function getMockAgentFilterOptions")
    expect(source).toContain("from '@/types/agent.types'")
  })

  it("uses canonical Agent list query controls in mock data", () => {
    expect(source).toContain("params: AgentListQueryParams = {}")
    expect(source).toContain("params.pageToken")
    expect(source).toContain("params.filter")
    expect(source).toContain("params.orderBy")
    expect(source).toContain("totalSize")
    expect(source).toContain("nextPageToken")
    expect(source).not.toContain("status?: string")
    expect(source).not.toContain("status ? mockAgents.filter")
  })

  it("uses connection IP for Agent search while keeping location provenance separate", () => {
    expect(source).toContain('extractFilterValues(filter, "connectionIp")')
    expect(source).toContain("agent.connectionIp")
    expect(source).toContain("observedSourceIp: positionedLocation.sourceObservedIp")
    expect(source).not.toContain("agent.ipAddress")
  })

  it("owns deterministic operational fixtures without importing production map coordinates", () => {
    expect(source).toContain("export function getMockAgentClusterSummary")
    expect(source).toContain("export function getMockAgentLocationMap")
    expect(source).toContain("export function getMockRegistrationTokenById")
    expect(source).not.toContain("MOCK_AGENT_LOCATION_BY_ID")
    expect(source).not.toContain("MOCK_SERVER_LOCATION")
    expect(source).not.toContain("components/overview/agent-location-map-data")
  })
})
