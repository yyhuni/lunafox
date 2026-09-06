import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import { readdirSync, statSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/data-table/expandable-cell.tsx"), "utf8")
const componentsRoot = path.resolve(process.cwd(), "components")

function collectProductionTsxFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const fullPath = path.join(dir, entry)
    const stat = statSync(fullPath)

    if (stat.isDirectory()) {
      if (entry === "__tests__" || entry === "ui") return []
      return collectProductionTsxFiles(fullPath)
    }

    if (!entry.endsWith(".tsx")) return []
    if (fullPath.endsWith("components/shared/data-table/expandable-cell.tsx")) return []
    return [fullPath]
  })
}

describe("expandable-cell contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ExpandableI18nProvider")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("keeps long structured values in a three-line preview with a cell-local expander", () => {
    expect(source).toContain("maxLines = 3")
    expect(source).toContain("JSON.stringify(value, null, 2)")
    expect(source).toContain("el.scrollHeight > el.clientHeight + 1")
    expect(source).toContain("setExpanded(!expanded)")
    expect(source).toContain("line-clamp-3")
  })

  it("keeps badge-list preview on one line and hides overflow behind an explicit expander", () => {
    expect(source).toContain("singleLinePreview")
    expect(source).toContain("ResizeObserver")
    expect(source).toContain("whiteSpace: expanded ? \"normal\" : \"nowrap\"")
    expect(source).toContain("overflow: expanded ? \"visible\" : \"hidden\"")
    expect(source).toContain("visibility: \"hidden\"")
    expect(source).not.toContain('className={cn("flex flex-wrap items-center gap-1", className)}')
  })

  it("supports a bounded wrapping preview when a route must expose its configured item count", () => {
    expect(source).toContain("wrapPreview?: boolean")
    expect(source).toContain("expanded || wrapPreview ? \"flex flex-wrap\" : \"flex\"")
    expect(source).toContain("whiteSpace: expanded || wrapPreview ? \"normal\" : \"nowrap\"")
  })

  it("truncates long badge labels inside the cell width budget", () => {
    expect(source).toContain('className={cn(textRole.badgeSubtle, "min-w-0 max-w-full")}')
    expect(source).toContain('className="block min-w-0 truncate text-left"')
    expect(source).not.toContain('className="truncate text-left"')
  })

  it("supports table-cell typography roles without falling back to body copy rhythm", () => {
    expect(source).toContain('textRoleName?: "body" | "tableCellPrimary" | "tableCellSecondary"')
    expect(source).toContain("textRole[textRoleName]")
    expect(source).toContain('textRoleName.startsWith("tableCell") ? textRole.tableCellSecondary : textRole.bodySubtle')
  })

  it("requires production expandable table cells to choose an explicit text role", () => {
    const offenders = collectProductionTsxFiles(componentsRoot).flatMap((file) => {
      const fileSource = readFileSync(file, "utf8")
      const matches = fileSource.match(/<Expandable(?:Cell|UrlCell|MonoCell)\b(?![^>]*textRoleName=)[^>]*>/g)
      if (!matches) return []

      return matches.map((match) => `${path.relative(process.cwd(), file)}: ${match}`)
    })

    expect(offenders).toEqual([])
  })

  it("lets string tag lists opt into the same single-line badge preview owner as business-list badge cells", () => {
    expect(source).toContain("export interface ExpandableTagListProps")
    expect(source).toContain("singleLinePreview?: boolean")
    expect(source).toContain("maxVisible?: number")
    expect(source).toContain("<ExpandableBadgeList")
  })
})
