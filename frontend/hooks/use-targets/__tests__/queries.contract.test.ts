import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-targets/queries.ts"), "utf8")

describe("queries contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useTargets")
    expect(source).toContain("from \"@tanstack/react-query\"")
  })

  it("uses canonical target list controls in the query key and service call", () => {
    expect(source).toContain("pageSize: resolved.pageSize")
    expect(source).toContain("pageToken: resolved.pageToken")
    expect(source).toContain("filter: resolved.filter")
    expect(source).toContain("orderBy: resolved.orderBy")
    expect(source).toContain("getTargets({")
    expect(source).not.toContain("type: resolved.type")
    expect(source).not.toContain("page: resolved.page")
    expect(source).not.toContain("getTargets(resolved.page")
  })
})
