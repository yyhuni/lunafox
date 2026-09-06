import { describe, expect, it } from "vitest"
import {
  BUSINESS_LIST_DESKTOP_WIDTH_BUDGET_PX,
  getColumnResizeHeadroom,
  getColumnWidthContract,
  getDefaultColumnSizeTotal,
  readComponentSource,
} from "@/test/utils/business-list-width-contract"

const source = readComponentSource("components/endpoints/endpoints-columns.tsx")

function readColumnBlock(accessorKey: string) {
  const marker = `accessorKey: "${accessorKey}"`
  const start = source.indexOf(marker)
  expect(start).toBeGreaterThanOrEqual(0)
  const nextColumnStart = source.indexOf("\n    {", start + marker.length)
  return source.slice(start, nextColumnStart === -1 ? source.length : nextColumnStart)
}

describe("endpoints-columns contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function createEndpointColumns")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("groups HTTP status codes with their URL through the shared badge owner", () => {
    expect(source).toContain("HttpStatusBadge")
    expect(source).toContain('from "@/components/shared/status/http-status-badge"')
    expect(source).toContain('<div className="flex min-w-0 items-center gap-2">')
    expect(source).toContain('<HttpStatusBadge statusCode={row.original.statusCode} />')
    expect(source).toContain('<div className="min-w-0 flex-1">')
    expect(source).not.toContain('accessorKey: "statusCode"')
    expect(source).not.toContain('orderBy: "statusCode"')
    expect(source).not.toContain("getHttpStatusBadgeVariant")
    expect(source).not.toContain("getHttpStatusBadgeClassName")
    expect(source).not.toContain("const getStatusVariant")
    expect(source).not.toContain("text-emerald-600")
  })

  it("routes stable metric columns through the shared monospace owner", () => {
    expect(source).toContain("MonoValueCell")
    expect(source).toContain('accessorKey: "contentLength"')
    expect(source).toContain('accessorKey: "vhost"')
    expect(source).not.toContain('accessorKey: "responseTime"')
    expect(source).not.toContain(
      'return <span className={cn(textRole.tableCellSecondary, "font-mono tabular-nums")}>{new Intl.NumberFormat().format(len)}</span>'
    )
    expect(source).not.toContain(
      'return <span className={cn(textRole.tableCellSecondary, "font-mono")}>{vhost ? "true" : "false"}</span>'
    )
  })

  it("only exposes backend-approved endpoint columns as sortable", () => {
    expect(source).not.toContain('orderBy: "url"')
    expect(source).toContain('orderBy: "contentLength"')
    expect(source).toContain('orderBy: "createdAt"')
    expect(source).toContain('serverSortPerformance: "indexed"')

    for (const columnId of ["select", "url", "host", "title", "tech", "location", "webserver", "contentType", "responseBody", "responseHeaders", "vhost"]) {
      const column = columnId === "select" ? source.slice(source.indexOf('id: "select"'), source.indexOf('accessorKey: "url"')) : readColumnBlock(columnId)
      expect(column).not.toContain("orderBy")
      expect(column).toContain("enableSorting: false")
    }
  })

  it("uses compact semantic width bands for the default endpoint table columns", () => {
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

  it("keeps all visible endpoint columns within the desktop business-list width budget", () => {
    expect(getDefaultColumnSizeTotal(source)).toBeLessThanOrEqual(BUSINESS_LIST_DESKTOP_WIDTH_BUDGET_PX)
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
