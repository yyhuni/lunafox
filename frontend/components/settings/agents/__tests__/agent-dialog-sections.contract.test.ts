import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/agent-dialog-sections.tsx"), "utf8")

describe("agent-dialog-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function AgentConfigDialogHeader")
    expect(source).toContain("className")
    expect(source).toContain("from \"react-hook-form\"")
  })
})
