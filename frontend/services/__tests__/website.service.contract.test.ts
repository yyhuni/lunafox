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
import { WebsiteService } from "@/services/website.service"

const source = readFileSync(path.resolve(process.cwd(), "services/website.service.ts"), "utf8")

describe("website.service contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("preserves current source markers", () => {
    expect(source).toContain("from \"@/lib/api-client\"")
  })

  it("uses canonical backend query parameters for target websites", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [{
          id: 1,
          name: "targets/7/websites/1",
          url: "https://admin.example.com",
          createdAt: "2026-07-01T00:00:00Z",
          screenshot: {
            id: 3,
            name: "targets/7/screenshots/3",
            url: "https://admin.example.com",
            statusCode: 200,
            createdAt: "2026-07-01T00:00:00Z",
            updatedAt: "2026-07-01T00:01:00Z",
          },
        }],
        totalSize: 21,
        nextPageToken: "target-cursor-2",
      },
    } as never)

    const result = await WebsiteService.getTargetWebSites(7, {
      pageSize: 10,
      pageToken: "target-cursor-1",
      filter: 'url="admin" && (statusCode="200" || statusCode="301")',
      orderBy: "createdAt desc",
    })

    expect(api.get).toHaveBeenCalledWith("/targets/7/websites/", {
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
    expect(result.results[0]?.resourceName).toBe("targets/7/websites/1")
    expect(result.results[0]?.screenshot?.resourceName).toBe("targets/7/screenshots/3")
  })

  it("reads one Website through the flattened Get path and normalizes its canonical names", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        id: 1,
        name: "targets/7/websites/1",
        url: "https://admin.example.com",
        screenshot: {
          id: 3,
          name: "targets/7/screenshots/3",
          url: "https://admin.example.com",
          statusCode: 200,
          createdAt: "2026-07-01T00:00:00Z",
          updatedAt: "2026-07-01T00:01:00Z",
        },
      },
    } as never)

    const result = await WebsiteService.getWebsite(1)

    expect(api.get).toHaveBeenCalledWith("/websites/1/")
    expect(result.resourceName).toBe("targets/7/websites/1")
    expect(result.screenshot?.resourceName).toBe("targets/7/screenshots/3")
    expect(source).toContain("static async getWebsite")
    expect(source).not.toContain("websiteRelations")
  })

  it("uses canonical backend query parameters for scan websites", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [{ id: 11, url: "https://api.example.com", createdAt: "2026-07-01T00:00:00Z" }],
        totalSize: 12,
        nextPageToken: "scan-cursor-2",
      },
    } as never)

    const result = await WebsiteService.getScanWebSites(12, {
      pageSize: 20,
      pageToken: "scan-cursor-1",
      filter: 'url="api" && tech="nginx" && vhost="true"',
      orderBy: "contentLength desc",
    })

    expect(api.get).toHaveBeenCalledWith("/scans/12/websites/", {
      params: {
        pageSize: 20,
        pageToken: "scan-cursor-1",
        filter: 'url="api" && tech="nginx" && vhost="true"',
        orderBy: "contentLength desc",
      },
    })
    expect(vi.mocked(api.get).mock.calls[0]?.[1]).not.toMatchObject({
      params: expect.objectContaining({ page: expect.anything() }),
    })
    expect(result.total).toBe(12)
    expect(result.totalPages).toBe(1)
    expect(result.nextPageToken).toBe("scan-cursor-2")
  })

  it("reads parent-scoped website filter options through the canonical results envelope", async () => {
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { results: [{ value: "nginx", label: "nginx", count: 3 }] },
    } as never)
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { results: [{ value: "react", label: "react", count: 2 }] },
    } as never)

    const targetOptions = await WebsiteService.getTargetWebsiteFilterOptions(7, "webserver")
    const scanOptions = await WebsiteService.getScanWebsiteFilterOptions(12, "tech")

    expect(api.get).toHaveBeenNthCalledWith(1, "/targets/7/websites/filterOptions", {
      params: { field: "webserver" },
    })
    expect(api.get).toHaveBeenNthCalledWith(2, "/scans/12/websites/filterOptions", {
      params: { field: "tech" },
    })
    expect(targetOptions.results).toEqual([{ value: "nginx", label: "nginx", count: 3 }])
    expect(scanOptions.results).toEqual([{ value: "react", label: "react", count: 2 }])
  })

  it("uses canonical website batch and export paths", async () => {
    const blob = new Blob(["url\n"])
    vi.mocked(api.get).mockResolvedValue({ data: blob } as never)
    vi.mocked(api.post).mockResolvedValue({ data: { createdCount: 1, deletedCount: 1 } } as never)

    await WebsiteService.bulkDeleteWebSites(7, [1, 2])
    await WebsiteService.bulkCreateWebsites(7, ["https://a.example.com"])
    await WebsiteService.exportWebsitesByTargetId(7)
    await WebsiteService.exportWebsitesByScanId(12)

    expect(api.post).toHaveBeenNthCalledWith(1, "/websites:batchDelete", { names: ["targets/7/websites/1", "targets/7/websites/2"] })
    expect(api.post).toHaveBeenNthCalledWith(2, "/targets/7/websites:batchCreate", { urls: ["https://a.example.com"] })
    expect(api.get).toHaveBeenNthCalledWith(1, "/targets/7/websites/exportFiles/current", { responseType: "blob" })
    expect(api.get).toHaveBeenNthCalledWith(2, "/scans/12/websites/exportFiles/current", { responseType: "blob" })
  })
})
