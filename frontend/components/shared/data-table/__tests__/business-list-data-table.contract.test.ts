import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const businessListSource = readFileSync(
  path.resolve(process.cwd(), "components/shared/data-table/business-list-data-table.tsx"),
  "utf8"
)
const smartFilterBusinessListSource = readFileSync(
  path.resolve(process.cwd(), "components/shared/data-table/smart-filter-business-list-data-table.tsx"),
  "utf8"
)

describe("business-list-data-table contract", () => {
  it("preserves current source markers", () => {
    expect(businessListSource).toContain("export function BusinessListDataTable")
    expect(businessListSource).toContain("UnifiedDataTable")
    expect(smartFilterBusinessListSource).toContain("export function SmartFilterBusinessListDataTable")
    expect(smartFilterBusinessListSource).toContain("BusinessListDataTable")
  })

  it("bakes baseline business-list defaults into the shared route-facing wrapper", () => {
    expect(businessListSource).toContain("enableAutoColumnSizing: false")
    expect(businessListSource).toContain("| \"enableAutoColumnSizing\"")
    expect(businessListSource).toContain("columnLayout: \"fixed\"")
    expect(businessListSource).toContain("sortingMode: \"none\" as const")
    expect(businessListSource).toContain("toolbarDensity: \"compact\"")
    expect(businessListSource).toContain("| \"showColumnVisibility\"")
    expect(businessListSource).not.toContain("fillAvailableHeight")
    expect(businessListSource).toContain("...ui")
  })

  it("keeps smart-filter business-list routes on the shared baseline", () => {
    expect(smartFilterBusinessListSource).toContain("searchMode: \"smart\"")
    expect(smartFilterBusinessListSource).toContain("expandColumnIds")
    expect(smartFilterBusinessListSource).not.toContain("VisibilityState")
    expect(smartFilterBusinessListSource).not.toContain("columnVisibility")
    expect(smartFilterBusinessListSource).not.toContain("onColumnVisibilityChange")
    expect(smartFilterBusinessListSource).not.toContain("enableAutoColumnSizing: true")
  })
})
