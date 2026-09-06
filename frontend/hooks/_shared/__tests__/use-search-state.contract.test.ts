import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/_shared/use-search-state.ts"), "utf8")

describe("use-search-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useSearchState")
    expect(source).toContain("from \"react\"")
  })

  it("standardizes backend list search as debounced draft plus immediate commit", () => {
    expect(source).toContain("SEARCH_COMMIT_DEBOUNCE_MS = 300")
    expect(source).toContain("searchInput")
    expect(source).toContain("handleSearchInputChange")
    expect(source).toContain("commitSearch")
    expect(source).toContain("window.setTimeout")
    expect(source).toContain("window.clearTimeout")
    expect(source).not.toContain("handleSearchChange")
  })
})
