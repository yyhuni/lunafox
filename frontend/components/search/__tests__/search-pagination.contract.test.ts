import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/search/search-pagination.tsx"), "utf8")

describe("search-pagination contract", () => {
  it("uses cursor pagination rather than numbered totals", () => {
    expect(source).toContain("export function SearchPagination")
    expect(source).toContain('mode="cursor"')
    expect(source).toContain("canPreviousPage")
    expect(source).toContain("canNextPage")
    expect(source).not.toContain("totalPages")
    expect(source).not.toContain("onPageChange")
  })

  it("delegates compact pagination geometry to the shared pagination owner", () => {
    expect(source).toContain("SharedCompactPagination")
    expect(source).not.toContain("IconChevronLeft")
    expect(source).not.toContain("SelectTrigger")
  })
})
