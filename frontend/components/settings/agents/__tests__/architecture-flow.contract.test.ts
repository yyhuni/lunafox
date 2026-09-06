import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/architecture-flow.tsx"), "utf8")
const stateSource = readFileSync(path.resolve(process.cwd(), "components/settings/agents/architecture-flow-state.ts"), "utf8")

describe("architecture-flow contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ArchitectureFlow")
    expect(source).toContain("ArchitectureFlowShell")
    expect(source).toContain("ArchitectureFlowCanvas")
    expect(source).toContain("fillAvailableSpace")
    expect(source).toContain("showDiagramNote")
    expect(source).toContain('t("flowDiagramNote")')
    expect(source).not.toContain("from \"@xyflow/react\"")
  })

  it("keeps static topology data free from simulated runtime status", () => {
    expect(stateSource).not.toContain("status?: string")
    expect(stateSource).not.toContain('t("flowStatusOnline")')
    expect(stateSource).not.toContain('t("flowEngineRunning")')
  })
})
