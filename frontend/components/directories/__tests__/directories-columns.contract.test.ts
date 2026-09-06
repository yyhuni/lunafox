import { describe, expect, it } from "vitest"
import {
  getColumnResizeHeadroom,
  getColumnWidthContract,
  readComponentSource,
} from "@/test/utils/business-list-width-contract"

const source = readComponentSource("components/directories/directories-columns.tsx")

function readColumnBlock(accessorKey: string) {
  const marker = `accessorKey: "${accessorKey}"`
  const start = source.indexOf(marker)
  expect(start).toBeGreaterThanOrEqual(0)
  const nextColumnStart = source.indexOf("\n    {", start + marker.length)
  return source.slice(start, nextColumnStart === -1 ? source.length : nextColumnStart)
}

describe("directories-columns contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function createDirectoryColumns")
    expect(source).toContain("export function useDirectoryTableColumns")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("routes HTTP status codes through the shared badge owner", () => {
    expect(source).toContain("HttpStatusBadge")
    expect(source).toContain('from "@/components/shared/status/http-status-badge"')
    expect(source).not.toContain("getHttpStatusBadgeVariant")
    expect(source).not.toContain("getHttpStatusBadgeClassName")
    expect(source).not.toContain("bg-green-500")
    expect(source).not.toContain("bg-blue-500")
    expect(source).not.toContain("bg-yellow-500")
    expect(source).not.toContain("bg-red-500")
  })

  it("keeps primary and metric columns on explicit semantic width bands", () => {
    const url = getColumnWidthContract(source, "url")
    const status = getColumnWidthContract(source, "status")
    const contentLength = getColumnWidthContract(source, "contentLength")
    const contentType = getColumnWidthContract(source, "contentType")
    const createdAt = getColumnWidthContract(source, "createdAt")

    expect(getColumnResizeHeadroom(url)).toBeGreaterThanOrEqual(300)
    expect(status.maxSize).toBeLessThanOrEqual(120)
    expect(contentLength.maxSize).toBeLessThanOrEqual(150)
    expect(contentType.maxSize).toBeGreaterThanOrEqual(180)
    expect(createdAt.size).toBeGreaterThanOrEqual(176)
    expect(createdAt.minSize).toBeGreaterThanOrEqual(176)
    expect(createdAt.maxSize).toBeLessThanOrEqual(220)
  })

  it("routes content type through a shared single-badge owner instead of a raw page-local badge", () => {
    expect(source).toContain("SingleBadgeCell")
    expect(source).toContain("singleBadge: true")
    expect(source).not.toContain('<Badge variant="outline">{contentType}</Badge>')
  })

  it("renders content length without locale group separators", () => {
    expect(source).toContain('as string | null')
    expect(source).toContain('length ?? "-"')
    expect(source).not.toContain("String(length)")
    expect(source).not.toContain("Number(length)")
    expect(source).not.toContain("parseInt(length")
    expect(source).not.toContain("length.toLocaleString()")
    expect(source).not.toContain("new Intl.NumberFormat().format(length)")
  })

  it("only exposes backend-approved directory columns as sortable", () => {
    expect(source).toContain('orderBy: "status"')
    expect(source).toContain('orderBy: "contentLength"')
    expect(source).toContain('orderBy: "createdAt"')
    expect(source).toContain('serverSortPerformance: "indexed"')

    const selectColumn = source.slice(source.indexOf('id: "select"'), source.indexOf('accessorKey: "url"'))
    expect(selectColumn).not.toContain("orderBy")
    expect(selectColumn).not.toContain("serverSortPerformance")

    for (const columnId of ["url", "contentType"]) {
      const column = readColumnBlock(columnId)
      expect(column).not.toContain("orderBy")
      expect(column).not.toContain("serverSortPerformance")
    }
  })
})
