import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/data-table/table-toolbar.tsx"), "utf8")

describe("table-toolbar contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function TableToolbar")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses export terminology for generated table files", () => {
    expect(source).toContain("exportOptions")
    expect(source).toContain("renderExportButton")
    expect(source).toContain("IconFileExport")
    expect(source).toContain('tActions("export")')
    expect(source).not.toContain("downloadOptions")
    expect(source).not.toContain("renderDownloadButton")
    expect(source).not.toContain('tActions("download")')
  })

  it("passes toolbar density through shared table toolbar controls", () => {
    expect(source).toContain("toolbarDensity")
    expect(source).toContain('toolbarDensity = "compact"')
    expect(source).toContain('toolbarDensity === "standard" ? "default" : "sm"')
    expect(source).toContain("toolbarDensity={toolbarDensity}")
    expect(source).toContain("size={actionButtonSize}")
  })

  it("routes export and column visibility through shared menu owners", () => {
    expect(source).toContain("ToolbarActionMenu")
    expect(source).toContain("ColumnVisibilityMenu")
    expect(source).toContain("showColumnVisibility = true")
    expect(source).toContain("showColumnVisibility ?")
  })

  it("keeps icons on export triggers but not export dropdown items", () => {
    const dropdownItems = source.slice(source.indexOf("{exportOptions.map((option)"), source.indexOf("</ToolbarActionMenu>"))

    expect(source).toContain('{option.icon || <IconFileExport className="h-4 w-4" />}')
    expect(source).toContain('icon={<IconFileExport className="h-4 w-4" />}')
    expect(dropdownItems).toContain("{option.label}")
    expect(dropdownItems).not.toContain("option.icon")
    expect(dropdownItems).not.toContain("IconFileExport")
  })
})
