import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/data-table/single-badge-cell.tsx"), "utf8")

describe("single-badge-cell contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function SingleBadgeCell")
    expect(source).toContain("from \"@/components/ui/badge\"")
  })

  it("keeps one badge visible without wrapping and falls back to shared table secondary text", () => {
    expect(source).toContain("whitespace-nowrap")
    expect(source).toContain("min-w-0")
    expect(source).toContain("max-w-full")
    expect(source).toContain('cn("max-w-full truncate", badgeClassName)')
    expect(source).toContain("textRole.tableCellSecondary")
    expect(source).toContain("data-badge-type")
  })
})
