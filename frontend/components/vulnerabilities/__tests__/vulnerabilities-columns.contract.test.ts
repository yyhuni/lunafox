import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/vulnerabilities/vulnerabilities-columns.tsx"), "utf8")

function sourceBetween(startMarker: string, endMarker: string) {
  const start = source.indexOf(startMarker)
  const end = source.indexOf(endMarker)

  expect(start).toBeGreaterThanOrEqual(0)
  expect(end).toBeGreaterThan(start)

  return source.slice(start, end)
}

describe("vulnerabilities-columns contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function createVulnerabilityColumns")
    expect(source).toContain("className")
    expect(source).toContain("from \"@tanstack/react-table\"")
  })

  it("uses shared severity and review status helpers", () => {
    expect(source).toContain("getSeverityVariant")
    expect(source).toContain("getStatusToneTextClass")
    expect(source).toContain("t.severity[severity]")
    expect(source).not.toContain("const severityConfig")
    expect(source).not.toContain("bg-blue-500/10")
    expect(source).not.toContain("text-blue-600")
    expect(source).not.toContain("border-blue-500/30")
  })

  it("renders review status as a read-only icon indicator", () => {
    expect(source).toContain("getReviewStatusIndicatorClassName")
    expect(source).toContain("size: vulnerabilitiesTableColumnLayout.reviewStatus.size")
    expect(source).toContain('getStatusToneTextClass(isPending ? "muted" : "success")')
    expect(source).not.toContain("getStatusToneBadgeClass")
    expect(source).not.toContain('getStatusToneTextClass(isPending ? "info" : "muted")')
    expect(source).not.toContain("rounded-full")
    expect(source).not.toContain("bg-success/10")
    expect(source).not.toContain("bg-muted/10")
    expect(source).toContain('aria-label={isPending ? t.tooltips.pending : t.tooltips.reviewed}')
    expect(source).not.toContain("onToggleReview")
    expect(source).not.toContain('asChild\n              variant="outline"')
    expect(source).not.toContain('<button')
  })

  it("keeps compact support columns fixed while leaving width headroom for the controlled-variant primary text columns", () => {
    expect(source).toContain('from "./vulnerabilities-table-layout"')
    expect(source).toContain('id: "reviewStatus"')
    expect(source).toContain("size: vulnerabilitiesTableColumnLayout.reviewStatus.size")
    expect(source).toContain("maxSize: vulnerabilitiesTableColumnLayout.reviewStatus.maxSize")
    expect(source).toContain('accessorKey: "vulnType"')
    expect(source).toContain("maxSize: vulnerabilitiesTableColumnLayout.vulnType.maxSize")
    expect(source).toContain('accessorKey: "url"')
    expect(source).toContain("size: vulnerabilitiesTableColumnLayout.url.size")
    expect(source).toContain("maxSize: vulnerabilitiesTableColumnLayout.url.maxSize")
    expect(source).not.toContain('id: "actions"')
    expect(source).not.toContain("t.actions.details")
    expect(source).not.toContain("Eye")
  })

  it("leaves vulnerability type as text because the whole row opens details", () => {
    const vulnTypeColumn = sourceBetween('accessorKey: "vulnType"', 'accessorKey: "url"')

    expect(vulnTypeColumn).toContain('return vulnType')
    expect(vulnTypeColumn).not.toContain("handleViewDetail")
    expect(vulnTypeColumn).not.toContain("event.stopPropagation()")
    expect(vulnTypeColumn).not.toContain("data-row-click-exempt")
    expect(vulnTypeColumn).not.toContain("<Button")
    expect(vulnTypeColumn).not.toContain("<Tooltip")
  })

  it("exposes only createdAt as a backend sortable vulnerability column", () => {
    const severityColumn = sourceBetween('accessorKey: "severity"', 'accessorKey: "source"')
    const sourceColumn = sourceBetween('accessorKey: "source"', 'accessorKey: "vulnType"')
    const vulnTypeColumn = sourceBetween('accessorKey: "vulnType"', 'accessorKey: "url"')
    const urlColumn = sourceBetween('accessorKey: "url"', 'accessorKey: "createdAt"')
    const createdAtColumn = source.slice(source.indexOf('accessorKey: "createdAt"'))

    expect(createdAtColumn).toContain('orderBy: "createdAt"')
    expect(createdAtColumn).toContain("serverSortPerformance")
    expect(createdAtColumn).toContain('firstSortDirection: "desc"')
    expect(severityColumn).not.toContain("orderBy")
    expect(sourceColumn).not.toContain("orderBy")
    expect(vulnTypeColumn).not.toContain("orderBy")
    expect(urlColumn).not.toContain("orderBy")
  })
})
