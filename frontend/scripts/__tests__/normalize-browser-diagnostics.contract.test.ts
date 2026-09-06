import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "scripts/normalize-browser-diagnostics.mjs"), "utf8")

describe("normalize-browser-diagnostics contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from \"node:fs/promises\"")
  })
})
