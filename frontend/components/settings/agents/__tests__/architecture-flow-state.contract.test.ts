import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/architecture-flow-state.ts"), "utf8")

describe("architecture-flow-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function getSourceHandleStyle")
    expect(source).toContain("from \"react\"")
  })

  it("does not force react-flow handle utilities with important overrides", () => {
    expect(source).not.toContain("!h-2")
    expect(source).not.toContain("!w-2")
    expect(source).not.toContain("!border-0")
    expect(source).not.toContain("!bg-transparent")
  })

  it("uses shared chart role helpers instead of local raw chart utilities", () => {
    expect(source).toContain('from "@/lib/chart-config"')
    expect(source).toContain("getArchitectureFlowRoleIconClassName")
    expect(source).not.toContain('text-[color:var(--color-chart-2)]')
    expect(source).not.toContain('text-[color:var(--color-chart-1)]')
  })

  it("uses the project runtime roles in the architecture topology", () => {
    expect(source).toContain('RoleKind = "server" | "agent" | "engine"')
    expect(source).toContain("AGENT_COUNT")
    expect(source).toContain("ENGINE_EXECUTIONS_PER_AGENT")
    expect(source).toContain("flowAgentTitle")
    expect(source).toContain("flowEngineTitle")
    expect(source).not.toContain('"worker"')
    expect(source).not.toContain("WORKERS_PER_AGENT")
    expect(source).not.toContain("flowWorker")
    expect(source).not.toContain("agentNode")
    expect(source).not.toContain("executor")
  })
})
