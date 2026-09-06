import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "scripts/build-refactor-visual-baseline.mjs"), "utf8")

describe("build-refactor-visual-baseline contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from \"node:fs/promises\"")
  })
})
