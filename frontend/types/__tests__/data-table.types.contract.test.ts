import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "types/data-table.types.ts"), "utf8")

describe("data-table.types contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"@tanstack/react-table\"")
  })

  it("defines shared data-table toolbar density choices", () => {
    expect(source).toContain("DataTableToolbarDensity")
    expect(source).toContain("'compact' | 'standard'")
    expect(source).toContain("toolbarDensity?: DataTableToolbarDensity")
  })

  it("allows shared column metadata to pin critical table columns", () => {
    expect(source).toContain("stickyRight?: boolean")
  })

  it("allows categorical single-badge columns to reserve their first-frame width", () => {
    expect(source).toContain("singleBadgeValues?: readonly string[]")
  })
})
