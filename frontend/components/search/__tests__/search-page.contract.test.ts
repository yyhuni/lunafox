import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/search/search-page.tsx"), "utf8")

describe("search-page contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function SearchPage")
    expect(source).toContain("from \"./search-page-sections\"")
  })
})
