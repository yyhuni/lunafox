import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/data-table/use-simple-search.ts"), "utf8")

describe("use-simple-search contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useSimpleSearchState")
    expect(source).toContain("from \"react\"")
  })

  it("supports the shared debounced backend-list search contract", () => {
    expect(source).toContain("SEARCH_COMMIT_DEBOUNCE_MS = 300")
    expect(source).toContain("commitNow")
    expect(source).toContain("window.setTimeout")
    expect(source).toContain("window.clearTimeout")
    expect(source).not.toContain("submit")
    expect(source).not.toContain("setValue")
  })
})
