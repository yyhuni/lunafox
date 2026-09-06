import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "types/agent.types.ts"), "utf8")

describe("agent.types contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("/**")
  })

  it("defines authoritative operational projections without provider diagnostics", () => {
    expect(source).toContain("export interface AgentClusterSummary")
    expect(source).toContain("resourceName: 'agentClusterSummaries/current'")
    expect(source).toContain("export interface AgentLocationMap")
    expect(source).toContain("resourceName: 'agentLocationMaps/current'")
    expect(source).toContain("accuracyRadiusKm: number | null")
    expect(source).toContain("export interface RegistrationTokenResource")
    expect(source).toContain("observedIpGeneration: number")
    expect(source).toContain("connectionIp?: string")
    expect(source).not.toContain("ipAddress?: string")
    expect(source).not.toContain("lastAttemptAt")
    expect(source).not.toContain("failureClass")
  })
})
