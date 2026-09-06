import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/settings/agents/page.tsx"), "utf8")

describe("page contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export default function AgentPage")
    expect(source).toContain("<AgentList />")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/settings/agents/agent-list\"")
    expect(source).not.toContain("from \"next/dynamic\"")
    expect(source).not.toContain("loading: () => <AgentPageSkeleton")
  })
})
