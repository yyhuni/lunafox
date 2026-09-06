import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/data-table/simple-search-toolbar.tsx"), "utf8")

describe("simple-search-toolbar contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function SimpleSearchToolbar")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("supports the compact inline search style without a separate button", () => {
    expect(source).toContain('from "@/components/shared/search-input"')
    expect(source).toContain("SearchInput")
    expect(source).toContain("showInlineIcon")
    expect(source).toContain('showInlineIcon = true')
    expect(source).toContain("showButton")
    expect(source).toContain("showButton ?? false")
    expect(source).toContain("loading={loading && !shouldShowButton}")
  })

  it("keeps an explicit compact search submit button as a separate adjacent control", () => {
    expect(source).toContain("groupClassName")
    expect(source).toContain('"flex min-w-48 flex-1 items-center gap-2 sm:flex-none"')
    expect(source).not.toContain("data-search-group")
    expect(source).not.toContain("rounded-r-none border-r-0")
    expect(source).not.toContain("rounded-l-none border-l-0")
  })

  it("supports standard 36px toolbar density without local height overrides", () => {
    expect(source).toContain("toolbarDensity")
    expect(source).toContain('toolbarDensity = "compact"')
    expect(source).toContain('inputWidthMode?: "fixed" | "fill"')
    expect(source).toContain('inputWidthMode = "fixed"')
    expect(source).toContain("isStandardDensity")
    expect(source).toContain("defaultInputWidthClassName")
    expect(source).toContain('const fillInputWidthClassName = "w-full"')
    expect(source).toContain('const resolvedInputWidthClassName = inputWidthMode === "fill" ? fillInputWidthClassName : defaultInputWidthClassName')
    expect(source).toContain("toolbarDensity={toolbarDensity}")
    expect(source).toContain('size={isStandardDensity ? "icon" : "icon-sm"}')
    expect(source).toContain("aria-label={placeholder}")
    expect(source).not.toContain('!isStandardDensity && "h-8"')
  })

  it("keeps search usable when table toolbar controls wrap on mobile", () => {
    expect(source).toContain("flex w-full flex-wrap items-center gap-2 sm:w-auto")
    expect(source).toContain("flex min-w-48 flex-1 items-center gap-2 sm:flex-none")
    expect(source).toContain('const defaultInputWidthClassName = "w-full sm:w-72 lg:w-80"')
    expect(source).not.toContain("sm:w-44")
    expect(source).not.toContain("max-w-sm")
  })
})
