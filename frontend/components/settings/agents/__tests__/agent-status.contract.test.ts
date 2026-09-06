import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"
import {
  getAgentHealthTone,
  getAgentHeartbeatTextClass,
  getAgentStatusDistributionBarClass,
  getAgentStatusDistributionDotClass,
  getAgentStatusDistributionTone,
} from "@/components/settings/agents/agent-status"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/agent-status.ts"), "utf8")

describe("agent-status contract", () => {
  it("exports shared agent node status helpers", () => {
    expect(source).toContain('from "@/lib/agent-status-distribution"')
    expect(source).toContain("export function getAgentRuntimeStatus")
    expect(source).toContain("export function getAgentHealthClassName")
    expect(source).toContain("export function getAgentMetricBarClass")
    expect(source).toContain("export function getAgentMetricTextClass")
    expect(source).toContain("export function getAgentMetricAlertTextClass")
    expect(source).toContain("export function getAgentMetricTrackClass")
    expect(source).toContain("export function getAgentStatusStripBarClass")
    expect(source).toContain('export { getAgentStatusDistributionTone } from "@/lib/agent-status-distribution"')
    expect(source).toContain("export function getAgentStatusDistributionBarClass")
    expect(source).toContain("export function getAgentStatusDistributionDotClass")
    expect(source).toContain("export function getAgentRuntimeShellClass")
    expect(source).toContain("export function getAgentHeartbeatTextClass")
  })

  it("keeps heartbeat timestamp text neutral across runtime states", () => {
    expect(getAgentHeartbeatTextClass()).toBe("text-foreground")
  })

  it("treats error and critical health states as warning for agent nodes", () => {
    expect(getAgentHealthTone("error")).toBe("warning")
    expect(getAgentHealthTone("critical")).toBe("warning")
  })

  it("treats backend healthy health state as healthy for agent nodes", () => {
    expect(getAgentHealthTone("healthy")).toBe("success")
  })

  it("keeps agent status distribution colors owned by status helpers", () => {
    expect(getAgentStatusDistributionTone("healthy")).toBe("success")
    expect(getAgentStatusDistributionTone("warning")).toBe("warning")
    expect(getAgentStatusDistributionTone("offline")).toBe("error")
    expect(getAgentStatusDistributionTone("unknown")).toBe("muted")
    expect(getAgentStatusDistributionBarClass("offline")).toBe("bg-error")
    expect(getAgentStatusDistributionDotClass("offline")).toBe("bg-error")
  })
})
