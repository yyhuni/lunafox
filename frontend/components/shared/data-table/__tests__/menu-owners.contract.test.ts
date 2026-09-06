import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/data-table/menu-owners.tsx"), "utf8")

describe("menu-owners contract", () => {
  it("exports shared owners for toolbar, row, and column-visibility menus", () => {
    expect(source).toContain("export function ToolbarActionMenu")
    expect(source).toContain("export function DenseRowActionMenu")
    expect(source).toContain("export function ColumnVisibilityMenu")
  })

  it("keeps short toolbar and row menus on the shared content-fit dropdown policy", () => {
    expect(source).toContain('menuWidth = "content-fit"')
    expect(source).toContain('width="content-fit"')
    expect(source).not.toContain('className="w-48"')
    expect(source).not.toContain('className="w-40"')
  })

  it("keeps dense row action menus on the approved shared trigger treatment", () => {
    expect(source).toContain('variant="ghost"')
    expect(source).toContain('triggerSize = "icon-sm"')
    expect(source).toContain("<DenseRowActionOwner")
    expect(source).toContain('size={triggerSize}')
    expect(source).toContain("<MoreHorizontal className=\"h-4 w-4\"")
  })

  it("keeps column visibility menus in the shared helper layer", () => {
    expect(source).toContain("<IconLayoutColumns className=\"h-4 w-4\"")
    expect(source).toContain("DropdownMenuCheckboxItem")
    expect(source).toContain("column.toggleVisibility(!!value)")
  })
})
