import { describe, expect, it } from "vitest"
import {
  BUSINESS_LIST_DESKTOP_WIDTH_BUDGET_PX,
  getColumnResizeHeadroom,
  getColumnWidthContract,
  getDefaultColumnSizeTotal,
  readComponentSource,
} from "@/test/utils/business-list-width-contract"

const source = readComponentSource("components/websites/websites-columns.tsx")

function readColumnBlock(accessorKey: string) {
  const marker = `accessorKey: "${accessorKey}"`
  const start = source.indexOf(marker)
  expect(start).toBeGreaterThanOrEqual(0)
  const nextColumnStart = source.indexOf("\n    {", start + marker.length)
  return source.slice(start, nextColumnStart === -1 ? source.length : nextColumnStart)
}

describe("websites-columns contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function createWebSiteColumns")
    expect(source).toContain("export function useWebSiteTableColumns")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses compact semantic width bands for the default website table columns", () => {
    expect(source).toContain('accessorKey: "url"')
    expect(source).toContain("size: 220")
    expect(source).toContain("minSize: 152")
    expect(source).toContain('accessorKey: "host"')
    expect(source).toContain("size: 104")
    expect(source).toContain("maxSize: 150")
    expect(source).toContain('accessorKey: "title"')
    expect(source).toContain("size: 144")
    expect(source).toContain("maxSize: 260")
    expect(source).toContain('accessorKey: "tech"')
    expect(source).toContain("maxSize: 240")
  })

  it("declares URL, title, and technology flex behavior beside their width bounds", () => {
    expect(readColumnBlock("url")).toContain('widthPolicy: { mode: "flex", flex: 2, fill: true }')
    expect(readColumnBlock("title")).toContain('widthPolicy: { mode: "flex", flex: 1.25 }')
    expect(readColumnBlock("tech")).toContain('widthPolicy: { mode: "flex", flex: 0.75 }')
  })

  it("keeps all visible website columns within the desktop business-list width budget", () => {
    expect(getDefaultColumnSizeTotal(source)).toBeLessThanOrEqual(BUSINESS_LIST_DESKTOP_WIDTH_BUDGET_PX)
  })

  it("binds website table body copy to shared table typography roles", () => {
    expect(source).toContain('textRoleName="tableCellPrimary"')
    expect(source).toContain('textRoleName="tableCellSecondary"')
    expect(source).toContain("textRole.tableCellSecondary")
    expect(source).not.toContain('return <div className="text-sm">')
  })

  it("groups HTTP status codes with their URL through the shared badge owner", () => {
    expect(source).toContain("HttpStatusBadge")
    expect(source).toContain('from "@/components/shared/status/http-status-badge"')
    expect(source).toContain('<div className="flex min-w-0 items-center gap-2">')
    expect(source).toContain('<HttpStatusBadge statusCode={row.original.statusCode} />')
    expect(source).toContain('<div className="min-w-0 flex-1">')
    expect(source).not.toContain('accessorKey: "statusCode"')
    expect(source).not.toContain('orderBy: "statusCode"')
    expect(source).not.toContain('let variant: "default" | "secondary" | "destructive" | "outline"')
    expect(source).not.toContain("statusCode >= 200")
  })

  it("keeps the technology badge cell on the shared single-line preview pattern", () => {
    expect(source).toContain("ExpandableTagList")
    expect(source).toContain("singleLinePreview")
    expect(source).toContain("maxVisible={3}")
  })

  it("keeps full response payloads available as optional columns", () => {
    expect(source).toContain('accessorKey: "responseBody"')
    expect(source).toContain('accessorKey: "responseHeaders"')
  })

  it("only exposes backend-approved website columns as sortable", () => {
    expect(source).toContain('orderBy: "contentLength"')
    expect(source).toContain('orderBy: "createdAt"')
    expect(source).toContain('serverSortPerformance: "indexed"')

    for (const columnId of ["select", "url", "host", "title", "tech", "location", "webserver", "contentType", "responseBody", "responseHeaders", "vhost"]) {
      const column = columnId === "select" ? source.slice(source.indexOf('id: "select"'), source.indexOf('accessorKey: "url"')) : readColumnBlock(columnId)
      expect(column).not.toContain("orderBy")
      expect(column).not.toContain("serverSortPerformance")
    }
  })

  it("gives summary columns meaningful resize headroom instead of capping the drag near the compact default", () => {
    const tech = getColumnWidthContract(source, "tech")
    const url = getColumnWidthContract(source, "url")
    const contentLength = getColumnWidthContract(source, "contentLength")

    expect(getColumnResizeHeadroom(tech)).toBeGreaterThanOrEqual(100)
    expect(tech.maxSize).toBeGreaterThanOrEqual(220)
    expect(getColumnResizeHeadroom(url)).toBeGreaterThan(getColumnResizeHeadroom(contentLength))
    expect(contentLength.maxSize).toBeLessThanOrEqual(120)
  })
})
