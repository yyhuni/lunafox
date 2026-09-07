import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/agent-list-item.tsx"), "utf8")

describe("agent-list-item contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function AgentListItem")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses shared agent node status helpers instead of raw var utilities", () => {
    expect(source).toContain("getAgentHealthClassName")
    expect(source).toContain("getAgentMetricBarClass")
    expect(source).toContain("getAgentRuntimeStatus")
    expect(source).toContain("getAgentMetricAlertTextClass")
    expect(source).toContain("getAgentRuntimeShellClass")
    expect(source).toContain("getAgentHeartbeatTextClass")
    expect(source).not.toContain("[var(--success)]")
    expect(source).not.toContain("[var(--warning)]")
    expect(source).not.toContain("[var(--error)]")
    expect(source).not.toContain('status === "critical" ? "text-error" : "text-warning"')
    expect(source).not.toContain('agentNode.status === "offline" && "border-slate-300/50 opacity-75"')
  })

  it("uses the shared current-connection display instead of an IP fallback", () => {
    expect(source).toContain("getAgentConnectionIpDisplay")
    expect(source).not.toContain("agentNode.ipAddress")
    expect(source).not.toContain('t("unknownIp")')
  })

  it("routes the agent node list action menu through the shared dense row owner", () => {
    expect(source).toContain("DenseRowActionMenu")
    expect(source).toContain('ownerClassName="shrink-0"')
  })

  it("keeps inline metric bars on transform-based progress motion", () => {
    expect(source).toContain("origin-left")
    expect(source).toContain("scaleX")
    expect(source).not.toContain("transition-[width")
    expect(source).not.toContain("width: `${percentage}%`")
  })

  it("keeps running tasks separate from occupied task slots", () => {
    expect(source).toContain('t("metrics.runningTasks")')
    expect(source).toContain("heartbeat.runningTasks")
    expect(source).toContain('t("metrics.usedTaskSlots")')
    expect(source).toContain("heartbeat.taskSlotsUsed")
    expect(source).toContain("agentNode.maxTasks")
  })

  it("does not use local loading-class pulse animation for runtime status", () => {
    expect(source).toContain("StatusIndicator")
    expect(source).toContain("IconActivity")
    expect(source).not.toContain("animate-pulse")
  })
})
