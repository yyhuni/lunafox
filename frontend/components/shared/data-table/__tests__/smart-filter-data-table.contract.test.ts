import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/data-table/smart-filter-data-table.tsx"), "utf8")

describe("smart-filter-data-table contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function SmartFilterDataTable")
    expect(source).toContain("from \"@tanstack/react-table\"")
  })

  it("passes export options through the grouped actions API", () => {
    expect(source).toContain("exportOptions")
    expect(source).not.toContain("downloadOptions")
  })

  it("exposes baseline business-list controls through shared preset props", () => {
    expect(source).toContain("expandColumnIds?: string[]")
    expect(source).toContain("SmartFilterBusinessListDataTable")
    expect(source).not.toContain("toolbarDensity?: DataTableToolbarDensity")
    expect(source).not.toContain("enableAutoColumnSizing?: boolean")
  })
})
