import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(
  path.resolve(process.cwd(), "components/shared/data-table/target-type-filter-select.tsx"),
  "utf8"
)

describe("target-type-filter-select contract", () => {
  it("keeps target-type options as plain text rows", () => {
    expect(source).toContain('value: "all"')
    expect(source).toContain('value: "domain"')
    expect(source).toContain('value: "ip"')
    expect(source).toContain('value: "cidr"')
    expect(source).toContain("<Filter className=\"h-4 w-4\" />")
    expect(source).toContain("{option.label}")
    expect(source).not.toContain("<Globe className=\"h-4 w-4\" />")
    expect(source).not.toContain("<Server className=\"h-4 w-4\" />")
    expect(source).not.toContain("<Network className=\"h-4 w-4\" />")
  })

  it("exposes a faceted multi-select variant for target list mock filtering", () => {
    expect(source).toContain("export function TargetTypeFacetedFilter")
    expect(source).toContain("title: string")
    expect(source).toContain("values: TargetType[]")
    expect(source).toContain("onValuesChange: (values: TargetType[]) => void")
    expect(source).toContain("clearLabel: string")
    expect(source).toContain('import { DataTableFacetedFilter, DataTableFacetedFilterGroup } from "./faceted-filter"')
    expect(source).toContain("<DataTableFacetedFilterGroup")
    expect(source).toContain("hasSelectedValues={values.length > 0}")
    expect(source).toContain("onReset={() => onValuesChange([])}")
    expect(source).toContain("<DataTableFacetedFilter")
    expect(source).toContain("emptyLabel={labels.all}")
    expect(source).not.toContain('variant="filterCount"')
    expect(source).toContain("{clearLabel}")
  })

  it("keeps the target-type menu on the shared compact select contract", () => {
    expect(source).toContain("<SelectTrigger size=\"sm\" className=\"w-28\">")
    expect(source).toContain("<SelectContent width=\"content-fit\">")
    expect(source).toContain("<SelectValue placeholder={labels.all} />")
  })
})
