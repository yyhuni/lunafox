import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "postcss.config.mjs"), "utf8")

describe("postcss.config contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("const config = {")
  })
})
