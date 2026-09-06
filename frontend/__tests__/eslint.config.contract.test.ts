import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "eslint.config.mjs"), "utf8")

describe("eslint.config contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from \"path\"")
  })
})
