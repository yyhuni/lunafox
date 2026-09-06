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
import { EndpointService } from "@/services/endpoint.service"

const source = readFileSync(path.resolve(process.cwd(), "services/endpoint.service.ts"), "utf8")

describe("endpoint.service contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("preserves current source markers", () => {
    expect(source).toContain("from \"@/lib/api-client\"")
  })

  it("uses canonical backend query parameters for target endpoints", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [{ id: 1, url: "https://admin.example.com", createdAt: "2026-07-01T00:00:00Z" }],
        totalSize: 21,
        nextPageToken: "target-cursor-2",
      },
    } as never)

    const result = await EndpointService.getEndpointsByTargetId(7, {
      pageSize: 10,
      pageToken: "target-cursor-1",
      filter: 'url="admin" && (statusCode="200" || statusCode="301")',
      orderBy: "createdAt desc",
    })

    expect(api.get).toHaveBeenCalledWith("/targets/7/endpoints/", {
      params: {
        pageSize: 10,
        pageToken: "target-cursor-1",
        filter: 'url="admin" && (statusCode="200" || statusCode="301")',
        orderBy: "createdAt desc",
      },
    })
    expect(vi.mocked(api.get).mock.calls[0]?.[1]).not.toMatchObject({
      params: expect.objectContaining({ page: expect.anything() }),
    })
    expect(result.total).toBe(21)
    expect(result.totalPages).toBe(3)
    expect(result.nextPageToken).toBe("target-cursor-2")
  })

  it("uses canonical backend query parameters for scan endpoints", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [{ id: 11, url: "https://api.example.com", createdAt: "2026-07-01T00:00:00Z" }],
        totalSize: 12,
        nextPageToken: "scan-cursor-2",
      },
    } as never)

    const result = await EndpointService.getEndpointsByScanId(12, {
      pageSize: 20,
      pageToken: "scan-cursor-1",
      filter: 'url="api" && tech="nginx" && vhost="true"',
      orderBy: "statusCode",
    })

    expect(api.get).toHaveBeenCalledWith("/scans/12/endpoints/", {
      params: {
        pageSize: 20,
        pageToken: "scan-cursor-1",
        filter: 'url="api" && tech="nginx" && vhost="true"',
        orderBy: "statusCode",
      },
    })
    expect(vi.mocked(api.get).mock.calls[0]?.[1]).not.toMatchObject({
      params: expect.objectContaining({ page: expect.anything() }),
    })
    expect(result.total).toBe(12)
    expect(result.totalPages).toBe(1)
    expect(result.nextPageToken).toBe("scan-cursor-2")
  })

  it("reads parent-scoped endpoint filter options through the canonical results envelope", async () => {
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { results: [{ value: "nginx", label: "nginx", count: 3 }] },
    } as never)
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { results: [{ value: "react", label: "react", count: 2 }] },
    } as never)

    const targetOptions = await EndpointService.getTargetEndpointFilterOptions(7, "webserver")
    const scanOptions = await EndpointService.getScanEndpointFilterOptions(12, "tech")

    expect(api.get).toHaveBeenNthCalledWith(1, "/targets/7/endpoints/filterOptions", {
      params: { field: "webserver" },
    })
    expect(api.get).toHaveBeenNthCalledWith(2, "/scans/12/endpoints/filterOptions", {
      params: { field: "tech" },
    })
    expect(targetOptions.results).toEqual([{ value: "nginx", label: "nginx", count: 3 }])
    expect(scanOptions.results).toEqual([{ value: "react", label: "react", count: 2 }])
  })

  it("uses canonical endpoint batch and export paths", async () => {
    const blob = new Blob(["url\n"])
    vi.mocked(api.get).mockResolvedValue({ data: blob } as never)
    vi.mocked(api.post).mockResolvedValue({ data: { createdCount: 1, deletedCount: 1 } } as never)

    await EndpointService.bulkCreateEndpoints(7, ["https://a.example.com/api"])
    await EndpointService.batchDeleteEndpoints({ targetId: 7, ids: [1, 2] })
    await EndpointService.exportEndpointsByTargetId(7)
    await EndpointService.exportEndpointsByScanId(12)

    expect(api.post).toHaveBeenNthCalledWith(1, "/targets/7/endpoints:batchCreate", { urls: ["https://a.example.com/api"] })
    expect(api.post).toHaveBeenNthCalledWith(2, "/endpoints:batchDelete", { names: ["targets/7/endpoints/1", "targets/7/endpoints/2"] })
    expect(api.get).toHaveBeenNthCalledWith(1, "/targets/7/endpoints/exportFiles/current", { responseType: "blob" })
    expect(api.get).toHaveBeenNthCalledWith(2, "/scans/12/endpoints/exportFiles/current", { responseType: "blob" })
  })
})
