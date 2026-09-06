import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scan-searchable-picker.tsx"), "utf8")

describe("scan searchable picker contract", () => {
  it("owns the common bounded overlay and search input", () => {
    expect(source).toContain("<Popover")
    expect(source).toContain("<CommandInput")
    expect(source).toContain('className="w-[var(--anchor-width)] overflow-hidden p-0"')
    expect(source).toContain('className="max-h-52 sm:max-h-72"')
    expect(source).toContain('className="h-auto min-h-10 overflow-hidden py-2 text-left font-normal"')
  })

  it("reports its open state without moving picker-specific behavior into the shared overlay", () => {
    expect(source).toContain('onOpenChange?: (open: boolean) => void')
    expect(source).toContain('onOpenChange?.(nextOpen)')
    expect(source).toContain('onOpenChange={handleOpenChange}')
  })

  it("supports an opt-in controlled server-search mode without changing local filtering defaults", () => {
    expect(source).toContain("searchValue?: string")
    expect(source).toContain("onSearchValueChange?: (value: string) => void")
    expect(source).toContain("shouldFilter = true")
    expect(source).toContain("<Command shouldFilter={shouldFilter}")
    expect(source).toContain("listRef?: React.Ref<HTMLDivElement>")
    expect(source).toContain("aria-busy={listBusy}")
  })

  it("closes after selecting an enabled result without closing for disabled results", () => {
    expect(source).toContain('closest("[data-command-item]:not([data-disabled=true])")')
    expect(source).toContain("handleOpenChange(false)")
  })
})
