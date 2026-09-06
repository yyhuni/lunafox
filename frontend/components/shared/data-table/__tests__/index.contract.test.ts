import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/data-table/index.ts"), "utf8")
const paginationSource = readFileSync(path.resolve(process.cwd(), "components/shared/data-table/pagination.tsx"), "utf8")

describe("index contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from \"./unified-data-table\"")
  })

  it("exports route-facing business-list wrappers", () => {
    expect(source).toContain("BusinessListDataTable")
    expect(source).toContain("SmartFilterBusinessListDataTable")
  })

  it("exports the table export helper and option type", () => {
    expect(source).toContain("buildExportOptions")
    expect(source).toContain("ExportOption")
    expect(source).not.toContain("buildDownloadOptions")
    expect(source).not.toContain("DownloadOption")
  })

  it("exports shared dense row actions and quiet copy helpers", () => {
    expect(source).toContain("DenseRowActionOwner")
    expect(source).toContain("QuietCopyButton")
    expect(source).toContain("DENSE_ROW_ACTION_HOVER_REVEAL_CLASSNAME")
  })

  it("exports shared dropdown menu owners for business-list toolbars and row actions", () => {
    expect(source).toContain("ToolbarActionMenu")
    expect(source).toContain("DenseRowActionMenu")
    expect(source).toContain("ColumnVisibilityMenu")
  })

  it("exports the shared timestamp cell owner", () => {
    expect(source).toContain("TimestampCell")
    expect(source).toContain('from "./timestamp-cell"')
  })

  it("exports the shared monospace metric cell owner", () => {
    expect(source).toContain("MonoValueCell")
    expect(source).toContain('from "./mono-value-cell"')
  })

  it("exports the shared single-badge cell owner", () => {
    expect(source).toContain("SingleBadgeCell")
    expect(source).toContain('from "./single-badge-cell"')
  })

  it("exports the shared faceted filter owner", () => {
    expect(source).toContain("DataTableFacetPanel")
    expect(source).toContain("DataTableFacetedFilter")
    expect(source).toContain("DataTableFacetedFilterGroup")
    expect(source).toContain('from "./faceted-filter"')
    expect(source).toContain("DataTableFacetedFilterOption")
    expect(source).toContain("DataTableFacetedFilterContentSize")
    expect(source).toContain("DataTableFacetPanelFacet")
    expect(source).not.toContain("DataTableActiveFilter")
  })

  it("exports standardized business-list query helpers", () => {
    expect(source).toContain('from "./business-list-query"')
    expect(source).toContain("createBusinessListQuery")
    expect(source).toContain("compileBusinessListFilter")
    expect(source).toContain("toggleBusinessListSorting")
    expect(source).toContain("BusinessListQuery")
  })

  it("keeps pagination select width owned by the shared data-table module", () => {
    expect(paginationSource).toContain("SharedCompactPagination")
    expect(paginationSource).toContain('className="w-24"')
    expect(paginationSource).toContain('size="icon-sm"')
    expect(paginationSource).not.toContain("w-[90px]")
  })
})
