import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "types/target.types.ts"), "utf8")

describe("target.types contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("/**")
  })

  it("uses organizationIds for batch target creation", () => {
    expect(source).toContain("organizationIds?: number[]")
    expect(source).not.toContain("organizationId?: number")
  })
})
