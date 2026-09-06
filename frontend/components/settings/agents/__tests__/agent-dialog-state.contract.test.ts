import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/agent-dialog-state.ts"), "utf8")

describe("agent-dialog-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useAgentConfigDialogState")
    expect(source).toContain("from \"react\"")
  })
})
