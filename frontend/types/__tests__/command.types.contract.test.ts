import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "types/command.types.ts"), "utf8")

describe("command.types contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from \"./tool.types\"")
  })
})
