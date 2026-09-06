import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/table.tsx"), "utf8")

describe("table contract", () => {
  it("keeps default table framing and typography in the component", () => {
    expect(source).toContain("bg-secondary")
    expect(source).toContain('from "@/lib/typography"')
    expect(source).toContain("textRole.tableHeader")
    expect(source).not.toContain("tracking-[0.14em]")
    expect(source).not.toContain("uppercase")
    expect(source).toContain("bg-card")
    expect(source).toContain("hover:bg-secondary")
    expect(source).toContain("border-border")
    expect(source).toContain('TABLE_HEADER_RHYTHM_CLASS = "h-10 px-2"')
    expect(source).toContain('TABLE_CELL_RHYTHM_CLASS = "p-2"')
    expect(source).toContain("TABLE_CELL_RHYTHM_CLASS")
    expect(source).toContain("align-middle")
  })

  it("keeps the visible header row on the shared header surface", () => {
    expect(source).toContain("[&_tr]:bg-secondary")
  })

  it("exports shared table row rhythms for dense and identity-led business lists", () => {
    expect(source).toContain('TABLE_DENSE_ROW_CLASS = "h-12"')
    expect(source).toContain('TABLE_DENSE_CELL_RHYTHM_CLASS = "px-2 py-1"')
    expect(source).toContain("TABLE_DENSE_ROW_RHYTHM_HEIGHT_PX = 48")
    expect(source).toContain("TABLE_DENSE_ROW_ESTIMATED_HEIGHT_PX = 49")
    expect(source).toContain('TABLE_COMFORTABLE_ROW_CLASS = "h-16"')
    expect(source).toContain('TABLE_COMFORTABLE_CELL_RHYTHM_CLASS = "px-2 py-2"')
    expect(source).toContain("TABLE_COMFORTABLE_ROW_RHYTHM_HEIGHT_PX = 64")
    expect(source).toContain("TABLE_COMFORTABLE_ROW_ESTIMATED_HEIGHT_PX = 65")
    expect(source).toContain("TABLE_HEADER_RHYTHM_HEIGHT_PX = 40")
    expect(source).toContain("usesSharedTableRhythm")
    expect(source).toContain("!usesSharedTableRhythm && TABLE_CELL_RHYTHM_CLASS")
  })
})
