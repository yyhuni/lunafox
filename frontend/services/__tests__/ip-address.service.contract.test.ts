import { beforeEach, describe, expect, it, vi } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const apiMocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock("@/lib/api-client", () => ({
  api: apiMocks,
}))

import { api } from "@/lib/api-client"
import { IPAddressService } from "@/services/ip-address.service"

const source = readFileSync(path.resolve(process.cwd(), "services/ip-address.service.ts"), "utf8")

describe("ip-address.service contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("preserves current source markers", () => {
    expect(source).toContain("from \"@/lib/api-client\"")
  })

  it("uses canonical backend query parameters for target and scan hostPorts", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: { results: [], totalSize: 0 } } as never)

    await IPAddressService.getTargetIPAddresses(7, {
      pageSize: 25,
      pageToken: "target-page-2",
      filter: '(ip="192.168" || host="api") && (port="80" || port="443")',
      orderBy: "createdAt desc",
    })
    await IPAddressService.getScanIPAddresses(12, {
      pageSize: 10,
      pageToken: "scan-page-2",
      filter: 'host="api"',
      orderBy: "ip",
    })

    expect(api.get).toHaveBeenNthCalledWith(1, "/targets/7/hostPorts", {
      params: {
        pageSize: 25,
        pageToken: "target-page-2",
        filter: '(ip="192.168" || host="api") && (port="80" || port="443")',
        orderBy: "createdAt desc",
      },
    })
    expect(api.get).toHaveBeenNthCalledWith(2, "/scans/12/hostPorts", {
      params: {
        pageSize: 10,
        pageToken: "scan-page-2",
        filter: 'host="api"',
        orderBy: "ip",
      },
    })
    expect(vi.mocked(api.get).mock.calls[0]?.[1]).not.toMatchObject({
      params: expect.objectContaining({ page: expect.anything() }),
    })
    expect(vi.mocked(api.get).mock.calls[1]?.[1]).not.toMatchObject({
      params: expect.objectContaining({ page: expect.anything() }),
    })
  })

  it("uses canonical hostPorts export and batch delete paths", async () => {
    const blob = new Blob(["ip,host,port\n"])
    vi.mocked(api.get).mockResolvedValue({ data: blob } as never)
    vi.mocked(api.post).mockResolvedValue({ data: { deletedCount: 2 } } as never)

    await IPAddressService.exportIPAddressesByTargetId(7, ["192.0.2.1"])
    await IPAddressService.exportIPAddressesByScanId(12)
    await IPAddressService.bulkDelete(["192.0.2.1", "192.0.2.2"])

    expect(api.get).toHaveBeenNthCalledWith(1, "/targets/7/hostPorts/exportFiles/current", {
      params: { ips: "192.0.2.1" },
      responseType: "blob",
    })
    expect(api.get).toHaveBeenNthCalledWith(2, "/scans/12/hostPorts/exportFiles/current", {
      responseType: "blob",
    })
    expect(api.post).toHaveBeenCalledWith("/hostPorts:batchDelete", {
      ips: ["192.0.2.1", "192.0.2.2"],
    })
  })

  it("reads parent-scoped hostPort port options through the canonical results envelope", async () => {
    vi.mocked(api.get).mockResolvedValueOnce({ data: { results: [{ value: "443", label: "443", count: 2 }] } } as never)
    vi.mocked(api.get).mockResolvedValueOnce({ data: { results: [{ value: "80", label: "80", count: 1 }] } } as never)

    const targetOptions = await IPAddressService.getTargetPortOptions(7)
    const scanOptions = await IPAddressService.getScanPortOptions(12)

    expect(api.get).toHaveBeenNthCalledWith(1, "/targets/7/hostPorts/filterOptions", {
      params: { field: "port" },
    })
    expect(api.get).toHaveBeenNthCalledWith(2, "/scans/12/hostPorts/filterOptions", {
      params: { field: "port" },
    })
    expect(targetOptions.results).toEqual([{ value: "443", label: "443", count: 2 }])
    expect(scanOptions.results).toEqual([{ value: "80", label: "80", count: 1 }])
  })

  it("normalizes AIP pagination for target hostPorts", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [{ ip: "192.0.2.1", hosts: ["a.example.com"], ports: [443], createdAt: "2026-06-01T00:00:00Z" }],
        totalSize: 21,
        nextPageToken: "page-2",
      },
    } as never)

    const result = await IPAddressService.getTargetIPAddresses(7, { pageSize: 10, pageToken: "cursor-1" })

    expect(result.total).toBe(21)
    expect(result.page).toBe(1)
    expect(result.pageSize).toBe(10)
    expect(result.totalPages).toBe(3)
    expect(result.nextPageToken).toBe("page-2")
  })

  it("normalizes backend-aggregated scan hostPorts without frontend row aggregation", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [
          { ip: "192.0.2.1", hosts: ["a.example.com", "b.example.com"], ports: [443, 8443], createdAt: "2026-06-01T00:00:00Z" },
        ],
        totalSize: 1,
        nextPageToken: "scan-cursor-2",
      },
    } as never)

    const result = await IPAddressService.getScanIPAddresses(12, { pageSize: 10 })

    expect(result.results).toEqual([
      {
        ip: "192.0.2.1",
        hosts: ["a.example.com", "b.example.com"],
        ports: [443, 8443],
        createdAt: "2026-06-01T00:00:00Z",
      },
    ])
    expect(result.total).toBe(1)
    expect(result.nextPageToken).toBe("scan-cursor-2")
    expect(source).not.toContain("new Map<string")
  })
})
