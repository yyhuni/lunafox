import { describe, expect, it } from "vitest"
import {
  getColumnResizeHeadroom,
  getColumnWidthContract,
  readComponentSource,
} from "@/test/utils/business-list-width-contract"

const source = readComponentSource("components/ip-addresses/ip-addresses-columns.tsx")

describe("ip-addresses-columns contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function createIPAddressColumns")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses shared single-line badge disclosure for the ports summary column", () => {
    expect(source).toContain("ExpandableBadgeList")
    expect(source).toContain('accessorKey: "ports"')
    expect(source).toContain("singleLinePreview")
    expect(source).not.toContain("PopoverTrigger")
    expect(source).not.toContain("flex flex-wrap gap-1.5 items-center")
  })

  it("keeps hosts and ports on bounded summary-width contracts", () => {
    const hosts = getColumnWidthContract(source, "hosts")
    const ports = getColumnWidthContract(source, "ports")
    const createdAt = getColumnWidthContract(source, "createdAt")

    expect(hosts.maxSize).toBeGreaterThanOrEqual(320)
    expect(ports.maxSize).toBeGreaterThanOrEqual(320)
    expect(getColumnResizeHeadroom(ports)).toBeGreaterThanOrEqual(100)
    expect(createdAt.size).toBeGreaterThanOrEqual(176)
    expect(createdAt.minSize).toBeGreaterThanOrEqual(176)
    expect(createdAt.maxSize).toBeLessThanOrEqual(220)
  })

  it("only exposes server-backed sorting for ip and createdAt", () => {
    expect(source).toContain("enableSorting: false")
    expect(source).toContain('orderBy: "ip"')
    expect(source).toContain('firstSortDirection: "asc"')
    expect(source).toContain("serverSortPerformance")
    expect(source).toContain("idx_hpm_target_ip")
    expect(source).toContain("idx_hpm_snap_scan_ip")
    expect(source).toContain('orderBy: "createdAt"')
    expect(source).toContain('firstSortDirection: "desc"')
    expect(source).toContain("idx_hpm_target_created_at")
    expect(source).toContain("idx_hpm_snap_scan_created_at")
    expect(source).toContain('accessorKey: "hosts"')
    expect(source).toContain('accessorKey: "ports"')
    expect(source).toContain("enableSorting: false")
  })
})
