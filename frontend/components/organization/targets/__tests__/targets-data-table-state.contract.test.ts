import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/targets/targets-data-table-state.ts"), "utf8")

describe("targets-data-table-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useOrganizationTargetsDataTableState")
    expect(source).toContain("from \"next-intl\"")
  })

  it("exposes shared table filter translations for the target-type faceted filter", () => {
    expect(source).toContain('const tColumns = useTranslations("columns")')
    expect(source).toContain('const tDataTable = useTranslations("dataTable")')
    expect(source).toContain("tColumns,")
    expect(source).toContain("tDataTable,")
  })
})
