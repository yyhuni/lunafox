import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/agent-expansion-slot.tsx"), "utf8")

describe("agent expansion slot contract", () => {
  it("uses one shared action surface instead of placeholder agent data", () => {
    expect(source).toContain("export function AgentExpansionSlot")
    expect(source).toContain('<Button')
    expect(source).toContain('variant="outline"')
    expect(source).toContain('border-dashed')
    expect(source).toContain("onClick={onOpenInstall}")
    expect(source).toContain("aria-label={title}")
    expect(source).toContain("AGENT_EXPANSION_SLOT_MIN_HEIGHT_CLASS")
    expect(source).not.toContain("AgentCardCompact")
  })

  it("occupies one grid cell with the same stable card geometry", () => {
    expect(source).not.toContain("min-h-[252px]")
    expect(source).toContain("rounded-lg")
    expect(source).not.toContain("col-span")
  })
})
