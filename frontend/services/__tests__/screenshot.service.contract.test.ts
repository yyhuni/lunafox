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
import { ScreenshotService } from "@/services/screenshot.service"

const source = readFileSync(path.resolve(process.cwd(), "services/screenshot.service.ts"), "utf8")

describe("screenshot.service contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("preserves current source markers", () => {
    expect(source).toContain("from \"@/lib/api-client\"")
  })

  it("uses canonical screenshot batch delete path", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { deletedCount: 2 } } as never)

    await ScreenshotService.bulkDelete(7, [1, 2])

    expect(api.post).toHaveBeenCalledWith("/screenshots:batchDelete", {
      names: ["targets/7/screenshots/1", "targets/7/screenshots/2"],
    })
  })

  it("builds canonical screenshot blob URLs", () => {
    expect(ScreenshotService.getImageUrl(7)).toBe("/v1/screenshots/7/blob")
    expect(ScreenshotService.getSnapshotImageUrl(12, 19)).toBe("/v1/scans/12/screenshotSnapshots/19/blob")
  })

  it("uses canonical backend query parameters for target screenshots", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [{ id: 1, url: "https://admin.example.com", statusCode: 200, createdAt: "2026-07-01T00:00:00Z" }],
        totalSize: 21,
        nextPageToken: "target-screenshot-cursor-2",
      },
    } as never)

    const result = await ScreenshotService.getByTarget(7, {
      pageSize: 12,
      pageToken: "target-screenshot-cursor-1",
      filter: 'url="admin" && (statusCode="200" || statusCode="301")',
      orderBy: "createdAt desc",
    })

    expect(api.get).toHaveBeenCalledWith("/targets/7/screenshots/", {
      params: {
        pageSize: 12,
        pageToken: "target-screenshot-cursor-1",
        filter: 'url="admin" && (statusCode="200" || statusCode="301")',
        orderBy: "createdAt desc",
      },
    })
    expect(vi.mocked(api.get).mock.calls[0]?.[1]).not.toMatchObject({
      params: expect.objectContaining({ page: expect.anything() }),
    })
    expect(result.total).toBe(21)
    expect(result.totalPages).toBe(2)
    expect(result.nextPageToken).toBe("target-screenshot-cursor-2")
  })

  it("uses canonical backend query parameters for scan screenshots", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [{ id: 11, url: "https://api.example.com", statusCode: 404, createdAt: "2026-07-01T00:00:00Z" }],
        totalSize: 13,
        nextPageToken: "scan-screenshot-cursor-2",
      },
    } as never)

    const result = await ScreenshotService.getByScan(12, {
      pageSize: 24,
      pageToken: "scan-screenshot-cursor-1",
      filter: 'url="api" && statusCode="404"',
      orderBy: "statusCode desc",
    })

    expect(api.get).toHaveBeenCalledWith("/scans/12/screenshots/", {
      params: {
        pageSize: 24,
        pageToken: "scan-screenshot-cursor-1",
        filter: 'url="api" && statusCode="404"',
        orderBy: "statusCode desc",
      },
    })
    expect(vi.mocked(api.get).mock.calls[0]?.[1]).not.toMatchObject({
      params: expect.objectContaining({ page: expect.anything() }),
    })
    expect(result.total).toBe(13)
    expect(result.totalPages).toBe(1)
    expect(result.nextPageToken).toBe("scan-screenshot-cursor-2")
  })

  it("requests target and scan screenshot filter options from scoped backend endpoints", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: { results: [{ value: "200", label: "200", count: 3 }] },
    } as never)

    const targetResult = await ScreenshotService.getTargetFilterOptions(7, "statusCode")
    const scanResult = await ScreenshotService.getScanFilterOptions(12, "statusCode")

    expect(api.get).toHaveBeenNthCalledWith(1, "/targets/7/screenshots/filterOptions", {
      params: { field: "statusCode" },
    })
    expect(api.get).toHaveBeenNthCalledWith(2, "/scans/12/screenshots/filterOptions", {
      params: { field: "statusCode" },
    })
    expect(targetResult.results[0]?.value).toBe("200")
    expect(scanResult.results[0]?.count).toBe(3)
  })
})
