import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-targets/keys.ts"), "utf8")

describe("keys contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from \"@/hooks/_shared/query-keys\"")
  })

  it("keys target lists by canonical backend query shape", () => {
    expect(source).toContain("list: (params: { pageSize?: number; pageToken?: string; filter?: string; orderBy?: string })")
    expect(source).not.toContain("list: (params: { page?:")
    expect(source).not.toContain("type?: string")
  })
})
