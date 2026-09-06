import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-history-data-table-sections.tsx"), "utf8")

describe("scan-history-data-table-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ScanHistoryToolbar")
    expect(source).toContain("SimpleSearchToolbar")
    expect(source).toContain("DataTableFacetedFilter")
  })

  it("uses the shared default inline-icon search field for scan history", () => {
    expect(source).toContain("SimpleSearchToolbar")
    expect(source).not.toContain("showInlineIcon")
    expect(source).not.toContain("showButton={false}")
    expect(source).not.toContain("inputClassName")
    expect(source).not.toContain("sm:w-[320px]")
  })

  it("reuses the shared faceted multi-select filter for scan status filtering", () => {
    expect(source).toContain('import { DataTableFacetedFilter, DataTableFacetedFilterGroup } from "@/components/shared/data-table/faceted-filter"')
    expect(source).toContain("status: ScanStatusFilter")
    expect(source).toContain("onStatusChange?: (status: ScanStatusFilter) => void")
    expect(source).toContain("<DataTableFacetedFilter<ScanStatus>")
    expect(source).toContain('title={state.tScan("statusLabel")}')
    expect(source).toContain("values={status}")
    expect(source).toContain("onValuesChange={onStatusChange}")
    expect(source).toContain("<DataTableFacetedFilterGroup")
    expect(source).toContain("hasSelectedValues={status.length > 0}")
    expect(source).toContain("onReset={() => onStatusChange([])}")
    expect(source).toContain("options={state.statusOptions}")
    expect(source).toContain('emptyLabel={state.tScan("allStatus")}')
    expect(source).toContain('clearLabel={state.tDataTable("clearFilter")}')
    expect(source).not.toContain("<SelectTrigger")
    expect(source).not.toContain("<SelectContent")
  })

  it("does not keep a route-local search-width override", () => {
    expect(source).not.toContain("inputClassName")
    expect(source).not.toContain("sm:w-[320px]")
  })
})
