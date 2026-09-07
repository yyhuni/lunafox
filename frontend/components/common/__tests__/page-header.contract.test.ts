import { describe, expect, it } from "vitest"
import { readdirSync, readFileSync, statSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/common/page-header.tsx"), "utf8")
const uiFoundationReadme = readFileSync(path.resolve(process.cwd(), "components/ui/README.md"), "utf8")

function collectSourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const filePath = path.join(dir, entry)
    const stat = statSync(filePath)

    if (stat.isDirectory()) {
      if (["__tests__", "node_modules"].includes(entry)) {
        return []
      }

      return collectSourceFiles(filePath)
    }

    return /\.(tsx|ts)$/.test(entry) ? [filePath] : []
  })
}

describe("page-header contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function PageHeader")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/lib/utils\"")
  })

  it("uses semantic typography roles for the display page title, code label, and description", () => {
    expect(source).toContain("textRole.pageTitleDisplay")
    expect(source).toContain("textRole.pageDescription")
    expect(source).toContain("textRole.monoLabel")
    expect(source).not.toContain("textRole.pageTitle}")
    expect(source).not.toContain("textRole.bodySubtle")
  })

  it("supports an optional bottom-aligned context slot without changing the default header layout", () => {
    expect(source).toContain("middle?: React.ReactNode")
    expect(source).toContain("const hasMiddle = Boolean(middle)")
    expect(source).toContain("items-end gap-2 xl:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)]")
    expect(source).toContain("xl:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)]")
    expect(source).toContain("hidden min-w-0 self-end items-center justify-self-center xl:flex")
    expect(source).toContain("col-start-2 self-end justify-self-end xl:col-start-3")
  })

  it("supports a responsive action aligned with the description row", () => {
    expect(source).toContain("descriptionAction?: React.ReactNode")
    expect(source).toContain("description || descriptionSupplement || descriptionAction")
    expect(source).toContain("flex flex-wrap items-center justify-between gap-x-4 gap-y-1")
    expect(source).toContain('className="ml-auto shrink-0"')
  })

  it("lets the page shell own the 16px header-to-content gap", () => {
    expect(source).toContain('className={cn("px-4 lg:px-6", className)}')
    expect(source).not.toContain('compact ? "mb-0" : "mb-2"')
    expect(source).toContain('description ? (compact ? "mb-1" : "mb-2") : "mb-0"')
  })

  it("does not require page-local margin overrides on PageHeader call sites", () => {
    const roots = ["app", "components"].map((root) => path.resolve(process.cwd(), root))
    const files = roots.flatMap(collectSourceFiles)
    const localMarginOverrides = files.flatMap((filePath) => {
      const fileSource = readFileSync(filePath, "utf8")
      const matches = fileSource.match(/<PageHeader\b[^>]*className=["'][^"']*\bmb-\d\b[^"']*["'][^>]*\/?>/g) ?? []

      return matches.map((match) => `${path.relative(process.cwd(), filePath)}: ${match}`)
    })

    expect(localMarginOverrides).toEqual([])
  })

  it("delegates responsive outer-shell rhythm to the shared route-shell contract", () => {
    expect(source).not.toContain("mb-4")
    expect(source).not.toContain("mb-6")
    expect(uiFoundationReadme).toContain("md:gap-6")
    expect(uiFoundationReadme).toContain("flex flex-col gap-4 py-4 md:gap-6 md:py-6")
  })

  it("documents the shared page header spacing contract", () => {
    expect(uiFoundationReadme).toContain("Page Header Spacing Standards")
    expect(uiFoundationReadme).toContain("16px")
    expect(uiFoundationReadme).toContain("gap-4")
    expect(uiFoundationReadme).toContain("md:gap-6")
    expect(uiFoundationReadme).toContain("PageHeader")
  })
})
