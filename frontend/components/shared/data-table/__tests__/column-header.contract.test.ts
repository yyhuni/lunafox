import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/data-table/column-header.tsx"), "utf8")

describe("column-header contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function DataTableColumnHeader")
    expect(source).toContain("className")
    expect(source).toContain("from \"@tanstack/react-table\"")
  })

  it("owns the shared compact status menu for sortable headers", () => {
    expect(source).toContain("from \"@/lib/typography\"")
    expect(source).toContain("textRole.tableHeader")
    expect(source).toContain('layout="tableHeaderInline"')
    expect(source).toContain("DropdownMenuContent")
    expect(source).toContain('width="compact"')
    expect(source).toContain('tDataTable("sortAsc")')
    expect(source).toContain('tDataTable("sortDesc")')
    expect(source).toContain("column.getCanHide()")
    expect(source).toContain("column.toggleVisibility(false)")
    expect(source).not.toContain("px-0")
    expect(source).not.toContain("has-[>svg]:px-0")
    expect(source).toContain("min-w-0 truncate")
  })
})
