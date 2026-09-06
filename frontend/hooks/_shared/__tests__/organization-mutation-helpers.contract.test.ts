import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/_shared/organization-mutation-helpers.ts"), "utf8")

describe("organization-mutation-helpers contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from \"@tanstack/react-query\"")
  })
})
