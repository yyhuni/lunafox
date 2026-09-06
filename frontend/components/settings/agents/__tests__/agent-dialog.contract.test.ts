import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/agent-dialog.tsx"), "utf8")

describe("agent-dialog contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function AgentConfigDialog")
    expect(source).toContain("className")
    expect(source).toContain("from \"next-intl\"")
  })
})
