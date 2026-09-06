import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/fingerprints/fingerprinthub-fingerprint-data-table.tsx"), "utf8")

describe("fingerprinthub-fingerprint-data-table contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function FingerPrintHubFingerprintDataTable")
    expect(source).toContain("from \"react\"")
  })

  it("uses the business-list baseline with simple search and canonical row identity", () => {
    expect(source).toContain("BusinessListDataTable")
    expect(source).toContain("SimpleSearchToolbar")
    expect(source).toContain("DataTableFacetedFilter")
    expect(source).toContain("DataTableFacetedFilterGroup")
    expect(source).toContain("query: BusinessListQuery")
    expect(source).toContain("getRowId={(row) => row.name}")
    expect(source).toContain('sortingMode: "server"')
    expect(source).toContain("paginationNavigation")
    expect(source).toContain('showAddButton: false')
    expect(source).toContain('filterOptions?: Partial<Record<"severity", FingerprintFilterOption[]>>')
    expect(source).not.toContain("FINGERPRINTHUB_SEVERITY_OPTIONS")
    expect(source).not.toContain('searchMode: "smart"')
    expect(source).not.toContain("SmartFilter")
    expect(source).not.toContain("onAddSingle")
    expect(source).not.toContain("DataTableFacetPanel")
  })

  it("exposes shared table loading controls for stateful page loading", () => {
    expect(source).toContain("loading?: boolean")
    expect(source).toContain("loadingRowCount?: number")
    expect(source).toContain("loading = false")
    expect(source).toContain("loadingRowCount,")
    expect(source).toContain("stableSurfaceRowCount,")
    expect(source).toContain("loading,")
    expect(source).toContain('loadingPresentation: initialLoading ? "initial" : "rows"')
  })
})
