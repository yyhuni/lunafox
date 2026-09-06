import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-history-data-table-state.ts"), "utf8")

describe("scan-history-data-table-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useScanHistoryDataTableState")
    expect(source).toContain("from \"next-intl\"")
  })

  it("keeps scan status options as real statuses for shared faceted filtering", () => {
    expect(source).toContain('const tDataTable = useTranslations("dataTable")')
    expect(source).toContain('import type { DataTableFacetedFilterOption } from "@/components/shared/data-table/faceted-filter"')
    expect(source).toContain("const statusOptions: Array<DataTableFacetedFilterOption<ScanStatus>>")
    expect(source).toContain('value: "running"')
    expect(source).toContain('value: "succeeded"')
    expect(source).toContain('value: "failed"')
    expect(source).toContain('value: "pending"')
    expect(source).toContain('value: "cancelled"')
    expect(source).not.toContain('value: "all"')
  })

  it("attaches scan-history summary counts to faceted status options", () => {
    expect(source).toContain('import { useScanStatistics } from "@/hooks/use-scans"')
    expect(source).toContain("const { data: scanStatistics } = useScanStatistics()")
    expect(source).toContain("count: scanStatistics?.running ?? 0")
    expect(source).toContain("count: scanStatistics?.succeeded ?? 0")
    expect(source).toContain("count: scanStatistics?.failed ?? 0")
    expect(source).toContain("count: scanStatistics?.pending ?? 0")
    expect(source).toContain("count: scanStatistics?.cancelled ?? 0")
  })
})
