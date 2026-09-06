import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "lib/domain-validator.ts"), "utf8")

describe("domain-validator contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from 'validator'")
  })
})
