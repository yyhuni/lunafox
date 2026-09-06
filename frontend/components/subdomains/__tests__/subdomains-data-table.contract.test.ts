import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/subdomains/subdomains-data-table.tsx"), "utf8")

describe("subdomains-data-table contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function SubdomainsDataTable")
    expect(source).toContain("BusinessListDataTable")
  })

  it("uses the standard simple search toolbar instead of SmartFilter for migrated subdomains", () => {
    expect(source).toContain("SimpleSearchToolbar")
    expect(source).toContain('placeholder={tActions("searchSubdomain")}')
    expect(source).not.toContain("SmartFilterBusinessListDataTable")
    expect(source).not.toContain("filterExamples")
    expect(source).not.toContain("filterFields")
  })

  it("allows migrated target and scan subdomain tables to run in server sorting mode", () => {
    expect(source).toContain("sortingMode?: DataTableSortingMode")
    expect(source).toContain('sortingMode = "none"')
    expect(source).toContain("sortingMode,")
    expect(source).toContain("sorting?: SortingState")
    expect(source).toContain("onSortingChange?: (sorting: SortingState) => void")
  })

  it("exposes shared table loading controls for stateful page loading", () => {
    expect(source).toContain("loading?: boolean")
    expect(source).toContain("initialLoading?: boolean")
    expect(source).toContain("loadingRowCount?: number")
    expect(source).toContain("loading = false")
    expect(source).toContain("initialLoading = false")
    expect(source).toContain("loadingRowCount,")
    expect(source).toContain("loading,")
    expect(source).toContain('loadingPresentation: initialLoading ? "initial" : "rows"')
  })
})
