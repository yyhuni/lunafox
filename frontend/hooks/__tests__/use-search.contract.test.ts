import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-search.ts"), "utf8")

describe("use-search contract", () => {
  it("only enables a query when the new global-search request shape is present", () => {
    expect(source).toContain("export function useAssetSearch")
    expect(source).toContain('from "@tanstack/react-query"')
    expect(source).toContain("params: SearchParams | undefined")
    expect(source).toContain('queryKey: ["asset-search", params ?? null]')
    expect(source).toContain("Boolean(params?.q.trim())")
    expect(source).not.toContain("asset_type")
    expect(source).not.toContain("totalPages")
  })
})
