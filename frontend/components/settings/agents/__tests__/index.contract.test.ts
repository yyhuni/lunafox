import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/index.ts"), "utf8")

describe("index contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from './agent-list'")
  })
})
