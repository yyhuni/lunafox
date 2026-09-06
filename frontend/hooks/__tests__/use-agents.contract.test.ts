import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-agents.ts"), "utf8")

describe("use-agents contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useAgents")
    expect(source).toContain("from '@tanstack/react-query'")
  })
})
