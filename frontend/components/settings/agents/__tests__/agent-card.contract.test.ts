import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/agent-card.tsx"), "utf8")

describe("agent-card contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function AgentCard")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses shared agent node status helpers instead of raw var utilities", () => {
    expect(source).toContain("getAgentHealthClassName")
    expect(source).toContain("getAgentRuntimeStatus")
    expect(source).toContain("getAgentRuntimeTextClass")
    expect(source).toContain("getAgentRuntimeShellClass")
    expect(source).toContain("getAgentHeartbeatTextClass")
    expect(source).not.toContain("[var(--success)]")
    expect(source).not.toContain("[var(--warning)]")
    expect(source).not.toContain("[var(--error)]")
    expect(source).not.toContain('agentNode.status === "offline" && "border-slate-300/50 opacity-75"')
  })

  it("uses the shared current-connection display instead of a GeoIP fallback", () => {
    expect(source).toContain("getAgentConnectionIpDisplay")
    expect(source).not.toContain("agentNode.ipAddress")
    expect(source).not.toContain("observedSourceIp")
  })

  it("routes the agent node action menu through the shared dense row owner", () => {
    expect(source).toContain("DenseRowActionMenu")
    expect(source).toContain('ownerClassName="shrink-0"')
  })

  it("keeps running tasks separate from occupied task slots", () => {
    expect(source).toContain('t("metrics.runningTasks")')
    expect(source).toContain("heartbeat.runningTasks")
    expect(source).toContain('t("metrics.usedTaskSlots")')
    expect(source).toContain("heartbeat.taskSlotsUsed")
    expect(source).toContain("agentNode.maxTasks")
  })

  it("uses the shared ordinary metric role for peer execution values", () => {
    expect(source).toContain("textRole.metricValueDisplay")
    expect(source).not.toContain('className="font-medium text-lg"')
  })

  it("does not use local loading-class pulse animation for runtime status", () => {
    expect(source).toContain("StatusIndicator")
    expect(source).toContain("IconActivity")
    expect(source).not.toContain("animate-pulse")
  })
})
