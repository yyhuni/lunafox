import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "mock/data/commands.ts"), "utf8")

describe("commands contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from \"@/types/command.types\"")
  })
})
