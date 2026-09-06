import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "lib/table-utils.ts"), "utf8")

describe("table-utils contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function measureTextWidth")
  })

  it("exports a shared header floor helper for sortable business-table columns", () => {
    expect(source).toContain('from "@/lib/font-stacks"')
    expect(source).toContain("export function getColumnHeaderMinWidthPx")
    expect(source).toContain("UI_SANS_MEASURE_FONT_14_MEDIUM")
    expect(source).toContain("TABLE_HEADER_CELL_HORIZONTAL_PADDING_PX")
    expect(source).toContain("TABLE_HEADER_TRIGGER_HORIZONTAL_PADDING_PX")
    expect(source).toContain("TABLE_HEADER_SORT_ICON_WIDTH_PX")
    expect(source).toContain("includeSortIcon")
    expect(source).toContain("TABLE_HEADER_MEASUREMENT_BUFFER_PX")
  })

  it("exports a shared badge floor helper for single-label table cells", () => {
    expect(source).toContain("export function getBadgeMinWidthPx")
    expect(source).toContain("UI_SANS_MEASURE_FONT_14")
    expect(source).toContain("TABLE_BADGE_TEXT_FONT")
    expect(source).toContain("UI_SANS_MEASURE_FONT_12_MEDIUM")
    expect(source).toContain("TABLE_BADGE_HORIZONTAL_PADDING_PX")
    expect(source).toContain("TABLE_BADGE_CELL_HORIZONTAL_PADDING_PX")
    expect(source).toContain("TABLE_BADGE_MEASUREMENT_BUFFER_PX")
  })
})
