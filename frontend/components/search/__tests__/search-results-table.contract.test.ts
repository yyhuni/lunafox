import { describe, expect, it } from "vitest"
import { readComponentSource } from "@/test/utils/business-list-width-contract"

const source = readComponentSource("components/search/search-results-table.tsx")

describe("search-results-table contract", () => {
  it("preserves the Endpoint table ownership", () => {
    expect(source).toContain("export function SearchResultsTable")
    expect(source).toContain("EndpointSearchResult")
  })

  it("keeps the preview table on fixed semantic widths with shared column controls and no pagination", () => {
    expect(source).toContain('columnLayout: "fixed"')
    expect(source).toContain('expandColumnIds: ["url", "title"]')
    expect(source).toContain("VisibilityState")
    expect(source).toContain("DEFAULT_SEARCH_COLUMN_VISIBILITY")
    expect(source).toContain("responseBody: false")
    expect(source).toContain("responseHeaders: false")
    expect(source).toContain("columnVisibility")
    expect(source).toContain("onColumnVisibilityChange")
    expect(source).toContain("showColumnVisibility: true")
    expect(source).toContain("hidePagination: true")
    expect(source).not.toContain("BusinessListDataTable")
  })

  it("reuses the shared single-line disclosure owner for technology summaries and bounds long secondary text", () => {
    expect(source).toContain("ExpandableTagList")
    expect(source).toContain("singleLinePreview")
    expect(source).toContain('id: "tech"')
    expect(source).toContain("maxSize: 260")
    expect(source).toContain('id: "responseBody"')
    expect(source).toContain("maxSize: 420")
  })

  it("routes HTTP status codes through the shared badge owner", () => {
    expect(source).toContain("HttpStatusBadge")
    expect(source).toContain('from "@/components/shared/status/http-status-badge"')
    expect(source).not.toContain("const getStatusVariant")
    expect(source).not.toContain("statusCode >= 200")
  })
})
