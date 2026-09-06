import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "types/psl.d.ts"), "utf8")

describe("psl.d contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function parse")
  })
})
