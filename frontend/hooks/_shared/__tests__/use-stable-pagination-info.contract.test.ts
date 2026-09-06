import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/_shared/use-stable-pagination-info.ts"), "utf8")

describe("use-stable-pagination-info contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from \"react\"")
  })
})
