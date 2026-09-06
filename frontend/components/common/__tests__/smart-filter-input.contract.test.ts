import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/common/smart-filter-input.tsx"), "utf8")

describe("smart-filter-input contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function SmartFilterInput")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/icons\"")
  })

  it("owns toolbar density through the shared smart filter component", () => {
    expect(source).toContain("toolbarDensity?: DataTableToolbarDensity")
    expect(source).toContain("density?: DataTableToolbarDensity")
    expect(source).toContain('const resolvedToolbarDensity = toolbarDensity ?? density ?? "compact"')
    expect(source).toContain('searchButtonMode = "icon"')
    expect(source).toContain('const resolvedSearchButtonSize = isStandardDensity ? "icon" : "icon-sm"')
    expect(source).toContain('showIcon={searchButtonMode === "inline"}')
    expect(source).toContain("size={resolvedSearchButtonSize}")
  })

  it("allows route-owned integrated search bars without replacing the shared component", () => {
    expect(source).toContain("fieldClassName?: string")
    expect(source).toContain("toolbarClassName?: string")
    expect(source).toContain("inputClassName?: string")
    expect(source).toContain("searchButtonLabel?: string")
    expect(source).toContain('searchButtonMode?: "inline" | "icon" | "attached-primary"')
    expect(source).toContain('searchButtonMode === "attached-primary"')
    expect(source).toContain('className="radius-none border-l border-l-primary-foreground/20 shadow-none focus-visible:z-10"')
    expect(source).toContain('searchButtonMode === "icon"')
    expect(source).toContain("onClick={() => setOpen(true)}")
  })

  it("uses Base UI popover geometry variables instead of Radix variables", () => {
    expect(source).toContain("--anchor-width")
    expect(source).not.toContain("--radix-popover-trigger-width")
  })
})
