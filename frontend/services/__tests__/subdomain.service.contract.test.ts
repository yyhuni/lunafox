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
import { SubdomainService } from "@/services/subdomain.service"

const source = readFileSync(path.resolve(process.cwd(), "services/subdomain.service.ts"), "utf8")

describe("subdomain.service contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("preserves current source markers", () => {
    expect(source).toContain("from \"@/lib/api-client\"")
  })

  it("uses canonical backend query parameters for scan subdomains", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [
          {
            id: 3266,
            scanId: 12,
            name: "scans/12/subdomainSnapshots/3266",
            dnsName: "es1.test.com",
            createdAt: "2026-06-18T07:12:58.575252Z",
          },
        ],
        nextPageToken: "cursor-2",
        totalSize: 1087,
      },
    } as never)

    const result = await SubdomainService.getSubdomainsByScanId(12, {
      pageSize: 10,
      pageToken: "scan-cursor-1",
      filter: 'dnsName="es"',
      orderBy: "dnsName desc",
    })

    expect(api.get).toHaveBeenCalledWith("/scans/12/subdomains/", {
      params: {
        pageSize: 10,
        pageToken: "scan-cursor-1",
        filter: 'dnsName="es"',
        orderBy: "dnsName desc",
      },
    })
    expect(vi.mocked(api.get).mock.calls[0]?.[1]).not.toMatchObject({
      params: expect.objectContaining({ page: expect.anything() }),
    })
    expect(result.results[0]).toMatchObject({
      id: 3266,
      name: "es1.test.com",
      dnsName: "es1.test.com",
    })
    expect(result.total).toBe(1087)
    expect(result.totalPages).toBe(109)
    expect(result.nextPageToken).toBe("cursor-2")
  })

  it("uses canonical backend query parameters for target subdomains", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [
          {
            id: 1068,
            name: "targets/1/subdomains/1068",
            dnsName: "api.test.com",
            createdAt: "2026-07-04T03:08:37Z",
          },
        ],
        totalSize: 1068,
        nextPageToken: "target-cursor-2",
      },
    } as never)

    const result = await SubdomainService.getSubdomainsByTargetId(1, {
      pageSize: 10,
      pageToken: "target-cursor-1",
      filter: 'dnsName="api"',
      orderBy: "dnsName desc",
    })

    expect(api.get).toHaveBeenCalledWith("/targets/1/subdomains/", {
      params: {
        pageSize: 10,
        pageToken: "target-cursor-1",
        filter: 'dnsName="api"',
        orderBy: "dnsName desc",
      },
    })
    expect(vi.mocked(api.get).mock.calls[0]?.[1]).not.toMatchObject({
      params: expect.objectContaining({ page: expect.anything() }),
    })
    expect(result.results[0]).toMatchObject({
      id: 1068,
      name: "api.test.com",
      dnsName: "api.test.com",
    })
    expect(result.total).toBe(1068)
    expect(result.totalPages).toBe(107)
    expect(result.nextPageToken).toBe("target-cursor-2")
  })

  it("uses canonical subdomain batch and export paths", async () => {
    const blob = new Blob(["dnsName\n"])
    vi.mocked(api.get).mockResolvedValue({ data: blob } as never)
    vi.mocked(api.post).mockResolvedValue({ data: { createdCount: 1, deletedCount: 1 } } as never)

    await SubdomainService.bulkCreateSubdomains(7, ["a.example.com"])
    await SubdomainService.bulkDeleteSubdomains(7, [1, 2])
    await SubdomainService.exportSubdomainsByTargetId(7)
    await SubdomainService.exportSubdomainsByScanId(12)

    expect(api.post).toHaveBeenNthCalledWith(1, "/targets/7/subdomains:batchCreate", { dnsNames: ["a.example.com"] })
    expect(api.post).toHaveBeenNthCalledWith(2, "/subdomains:batchDelete", { names: ["targets/7/subdomains/1", "targets/7/subdomains/2"] })
    expect(api.get).toHaveBeenNthCalledWith(1, "/targets/7/subdomains/exportFiles/current", { responseType: "blob" })
    expect(api.get).toHaveBeenNthCalledWith(2, "/scans/12/subdomains/exportFiles/current", { responseType: "blob" })
  })
})
