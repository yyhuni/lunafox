import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-vulnerabilities/keys.ts"), "utf8")

describe("keys contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from \"@/hooks/_shared/query-keys\"")
  })
})
