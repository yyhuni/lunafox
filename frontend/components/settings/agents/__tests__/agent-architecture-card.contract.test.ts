import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/agent-architecture-card.tsx"), "utf8")

describe("agent-architecture-card contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function AgentArchitectureCard")
    expect(source).toContain("className")
    expect(source).toContain("from \"next-intl\"")
  })
})
