import { describe, expect, it } from "vitest"
import {
  getColumnWidthContract,
  readComponentSource,
} from "@/test/utils/business-list-width-contract"

const source = readComponentSource("components/subdomains/subdomains-columns.tsx")

describe("subdomains-columns contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from \"@tanstack/react-table\"")
  })

  it("keeps the subdomain table on a primary-text plus timestamp width model", () => {
    const name = getColumnWidthContract(source, "name")
    const createdAt = getColumnWidthContract(source, "createdAt")

    expect(name.minSize).toBeGreaterThanOrEqual(250)
    expect(createdAt.size).toBeGreaterThanOrEqual(176)
    expect(createdAt.minSize).toBeGreaterThanOrEqual(176)
    expect(createdAt.maxSize).toBeLessThanOrEqual(220)
    expect(source).toContain("TimestampCell")
  })

  it("routes timestamp cells through the shared timestamp owner", () => {
    expect(source).toContain("TimestampCell")
    expect(source).not.toContain('textRole.tableCellSecondary, "whitespace-nowrap"')
  })

  it("only exposes server-backed sorting for dnsName and createdAt", () => {
    expect(source).toContain("enableSorting: false")
    expect(source).toContain('orderBy: "dnsName"')
    expect(source).toContain('firstSortDirection: "asc"')
    expect(source).toContain("serverSortPerformance")
    expect(source).toContain("idx_subdomain_snap_scan_dns_name_id")
    expect(source).toContain('orderBy: "createdAt"')
    expect(source).toContain('firstSortDirection: "desc"')
    expect(source).toContain("idx_subdomain_snap_scan_created_at_id")
  })
})
