import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/_shared/targets-helpers.ts"), "utf8")

describe("targets-helpers contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from \"@/types/target.types\"")
  })

  it("keeps target list query params on canonical pageToken/filter/orderBy fields", () => {
    expect(source).toContain("pageSize?: number")
    expect(source).toContain("pageToken?: string")
    expect(source).toContain("filter?: string")
    expect(source).toContain("orderBy?: string")
    expect(source).not.toContain("type?: string")
    expect(source).not.toContain("page?: number")
    expect(source).not.toContain("resolvedType")
  })
})
