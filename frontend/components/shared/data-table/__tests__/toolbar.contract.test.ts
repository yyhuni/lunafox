import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/data-table/toolbar.tsx"), "utf8")

describe("toolbar contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function DataTableToolbar")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("routes simple search through the shared inline search toolbar standard", () => {
    expect(source).toContain("toolbarDensity")
    expect(source).toContain("SimpleSearchToolbar")
    expect(source).toContain("toolbarDensity={toolbarDensity}")
    expect(source).toContain("placeholder={placeholder}")
    expect(source).toContain('inputWidthMode="fill"')
    expect(source).not.toContain('inputClassName="sm:w-full"')
    expect(source).not.toContain("aria-label={t('search')}")
  })

  it("wraps data-table toolbar controls on narrow viewports", () => {
    expect(source).toContain("flex flex-col gap-2 sm:flex-row")
    expect(source).toContain("sm:items-start")
    expect(source).not.toContain("sm:items-center")
    expect(source).toContain("flex w-full min-w-0 flex-wrap items-center gap-2")
    expect(source).toContain("flex w-full flex-wrap items-center gap-2")
    expect(source).not.toContain("space-x-2")
  })

  it("keeps default search width capped while allowing custom left toolbars to use available space", () => {
    expect(source).toContain("const leftContentClassName = cn(")
    expect(source).toContain('!leftContent && "sm:max-w-xl"')
    expect(source).toContain("className={leftContentClassName}")
    expect(source).not.toContain('className="flex w-full min-w-0 flex-wrap items-center gap-2 sm:max-w-xl sm:flex-1"')
  })
})
