import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "types/tool.types.ts"), "utf8")

describe("tool.types contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("// Tool type enum")
  })
})
