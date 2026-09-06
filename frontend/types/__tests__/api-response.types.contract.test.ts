import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "types/api-response.types.ts"), "utf8")

describe("api-response.types contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("// Common API response types")
  })
})
