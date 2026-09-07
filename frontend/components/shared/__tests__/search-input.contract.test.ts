import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/search-input.tsx"), "utf8")

describe("search-input contract", () => {
  it("owns the inline-icon search input geometry", () => {
    expect(source).toContain("export function SearchInput")
    expect(source).toContain('from "@/components/ui/input"')
    expect(source).toContain('from "@/components/shared/loading/spinner"')
    expect(source).toContain('type="search"')
    expect(source).toContain('showIcon = true')
    expect(source).toContain('toolbarDensity = "compact"')
    expect(source).toContain('size={isStandardDensity ? "default" : "sm"}')
    expect(source).toContain('showIcon && "pl-9"')
    expect(source).toContain('loading && "pr-9"')
    expect(source).toContain('left-3 top-1/2 size-4')
    expect(source).not.toContain("rounded-r-none")
    expect(source).not.toContain("border-r-0")
  })
})
