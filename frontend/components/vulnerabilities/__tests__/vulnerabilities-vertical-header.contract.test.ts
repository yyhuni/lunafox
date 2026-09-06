import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/vulnerabilities/vulnerabilities-vertical-header.tsx"), "utf8")

describe("vulnerabilities-vertical-header contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function VulnerabilitiesVerticalHeader")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("does not use important tab height or badge padding overrides", () => {
    expect(source).not.toContain("!h-11")
    expect(source).not.toContain("md:!h-7")
    expect(source).not.toContain("!py-0")
  })

  it("consumes semantic filter tabs and filter count badges", () => {
    expect(source).toContain('<TabsList variant="filter"')
    expect(source).toContain('<TabsTrigger variant="filter"')
    expect(source).toContain("TabsCountBadge")
    expect(source).not.toContain('variant="filterCount"')
    expect(source).not.toContain('className="bg-muted/50 border')
    expect(source).not.toContain('className="bg-background/50 border-0')
  })

  it("avoids local green and blue bulk action patches", () => {
    expect(source).toContain("getStatusToneInteractiveOutlineClass")
    expect(source).not.toContain("text-green-600")
    expect(source).not.toContain("hover:bg-green-50")
    expect(source).not.toContain("text-blue-600")
    expect(source).not.toContain("hover:bg-blue-50")
    expect(source).not.toContain("border-success/20 hover:bg-success/10 hover:text-success")
    expect(source).not.toContain("border-info/20 hover:bg-info/10 hover:text-info")
  })

  it("keeps pagination controls unseparated from the toolbar by a vertical border", () => {
    expect(source).not.toContain('className="border-l flex gap-1.5')
  })

  it("keeps the top control bar on one compact density", () => {
    expect(source).toContain("md:flex-row md:items-start md:justify-between")
    expect(source).not.toContain("md:flex-row md:items-center md:justify-between")
    expect(source).toContain('<TabsList variant="filter" size="sm"')
    expect(source).toContain('<TabsTrigger variant="filter" size="sm"')
    expect(source).toContain('from "@/components/shared/search-input"')
    expect(source).toContain("<SearchInput")
    expect(source).toContain('toolbarDensity="compact"')
    expect(source).toContain("SharedCompactPagination")
    expect(source).toContain("DataTableFacetPanel")
    expect(source).toContain("type DataTableFacetPanelFacet")
    expect(source).toContain("const severityFacetPanelItems: DataTableFacetPanelFacet[]")
    expect(source).toContain("facets={severityFacetPanelItems}")
    expect(source).toContain("activeCount={severityFilter.length}")
    expect(source).not.toContain("DataTableFacetedFilterGroup")
    expect(source).toContain('size="sm"')
    expect(source).toContain('size="icon-sm"')
    expect(source).not.toContain("md:h-8")
    expect(source).not.toContain("md:size-8")
    expect(source).not.toContain("md:data-[size=sm]:h-8")
    expect(source).not.toContain("data-[size=sm]:h-10")
  })

  it("uses shared tab and helper typography for filter labels and pagination support text", () => {
    expect(source).toContain('from "@/lib/typography"')
    expect(source).toContain("textRole.tab")
    expect(source).not.toContain('className="font-medium text-sm"')
    expect(source).not.toContain('text-[10px]')
  })
})
