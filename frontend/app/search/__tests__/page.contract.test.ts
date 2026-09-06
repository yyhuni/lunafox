import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/search/page.tsx"), "utf8")

describe("page contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export default function Search")
    expect(source).toContain("<SearchPage />")
    expect(source).toContain("from \"@/components/search/search-page\"")
  })

  it("directly imports the route-critical search surface instead of hiding it behind a null chunk fallback", () => {
    expect(source).not.toContain("lazyPage(")
    expect(source).not.toContain("next/dynamic")
    expect(source).not.toContain("loading: () => null")
    expect(source).not.toContain("PageSectionSkeleton")
  })
})
