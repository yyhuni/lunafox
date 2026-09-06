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
import { DirectoryService } from "@/services/directory.service"

const source = readFileSync(path.resolve(process.cwd(), "services/directory.service.ts"), "utf8")

function directoryWireRow(overrides: Record<string, unknown> = {}) {
  return {
    id: 1,
    name: "targets/7/directories/1",
    url: "https://admin.example.com",
    status: 200,
    contentLength: "123",
    contentType: "text/html",
    duration: "456",
    createdAt: "2026-07-01T00:00:00Z",
    ...overrides,
  }
}

describe("directory.service contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("preserves current source markers", () => {
    expect(source).toContain("from \"@/lib/api-client\"")
  })

  it("uses canonical backend query parameters for target directories", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [directoryWireRow()],
        totalSize: 21,
        nextPageToken: "target-directory-cursor-2",
      },
    } as never)

    const result = await DirectoryService.getTargetDirectories(7, {
      pageSize: 10,
      pageToken: "target-directory-cursor-1",
      filter: 'url="admin" && (status="200" || status="301") && contentType="text/html"',
      orderBy: "createdAt desc",
    })

    expect(api.get).toHaveBeenCalledWith("/targets/7/directories/", {
      params: {
        pageSize: 10,
        pageToken: "target-directory-cursor-1",
        filter: 'url="admin" && (status="200" || status="301") && contentType="text/html"',
        orderBy: "createdAt desc",
      },
    })
    expect(vi.mocked(api.get).mock.calls[0]?.[1]).not.toMatchObject({
      params: expect.objectContaining({ page: expect.anything() }),
    })
    expect(result.total).toBe(21)
    expect(result.totalPages).toBe(3)
    expect(result.nextPageToken).toBe("target-directory-cursor-2")
  })

  it("uses canonical backend query parameters for scan directories", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [directoryWireRow({ id: 11, name: "scans/12/directorySnapshots/11", url: "https://api.example.com" })],
        totalSize: 12,
        nextPageToken: "scan-directory-cursor-2",
      },
    } as never)

    const result = await DirectoryService.getScanDirectories(12, {
      pageSize: 20,
      pageToken: "scan-directory-cursor-1",
      filter: 'url="api" && contentType="application/json"',
      orderBy: "contentLength desc",
    })

    expect(api.get).toHaveBeenCalledWith("/scans/12/directories/", {
      params: {
        pageSize: 20,
        pageToken: "scan-directory-cursor-1",
        filter: 'url="api" && contentType="application/json"',
        orderBy: "contentLength desc",
      },
    })
    expect(vi.mocked(api.get).mock.calls[0]?.[1]).not.toMatchObject({
      params: expect.objectContaining({ page: expect.anything() }),
    })
    expect(result.total).toBe(12)
    expect(result.totalPages).toBe(1)
    expect(result.nextPageToken).toBe("scan-directory-cursor-2")
  })

  it("preserves nullable int64 strings exactly and removes obsolete source fields", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [
          directoryWireRow({
            contentLength: "9223372036854775807",
            duration: "0",
            words: 12,
            lines: 3,
            websiteUrl: "https://admin.example.com",
          }),
          directoryWireRow({ id: 2, contentLength: null, duration: null }),
        ],
        totalSize: 2,
      },
    } as never)

    const result = await DirectoryService.getTargetDirectories(7)

    expect(result.results[0]).toEqual({
      id: 1,
      url: "https://admin.example.com",
      status: 200,
      contentLength: "9223372036854775807",
      contentType: "text/html",
      duration: "0",
      createdAt: "2026-07-01T00:00:00Z",
    })
    expect(result.results[1]?.contentLength).toBeNull()
    expect(result.results[1]?.duration).toBeNull()
    expect(result.results[0]).not.toHaveProperty("words")
    expect(result.results[0]).not.toHaveProperty("lines")
    expect(result.results[0]).not.toHaveProperty("websiteUrl")
  })

  it.each(["contentLength", "duration"] as const)(
    "rejects numeric %s instead of coercing through JavaScript number",
    async (field) => {
      vi.mocked(api.get).mockResolvedValue({
        data: {
          results: [directoryWireRow({ [field]: 9_007_199_254_740_992 })],
          totalSize: 1,
        },
      } as never)

      await expect(DirectoryService.getTargetDirectories(7)).rejects.toThrow(
        `Invalid Directory ${field}`
      )
    }
  )

  it("requests target and scan directory filter options from scoped backend endpoints", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: { results: [{ value: "200", label: "200", count: 3 }] },
    } as never)

    const targetResult = await DirectoryService.getTargetDirectoryFilterOptions(7, "status")
    const scanResult = await DirectoryService.getScanDirectoryFilterOptions(12, "contentType")

    expect(api.get).toHaveBeenNthCalledWith(1, "/targets/7/directories/filterOptions", {
      params: { field: "status" },
    })
    expect(api.get).toHaveBeenNthCalledWith(2, "/scans/12/directories/filterOptions", {
      params: { field: "contentType" },
    })
    expect(targetResult.results[0]?.value).toBe("200")
    expect(scanResult.results[0]?.count).toBe(3)
  })

  it("uses canonical directory batch and export paths", async () => {
    const blob = new Blob(["url\n"])
    vi.mocked(api.get).mockResolvedValue({ data: blob } as never)
    vi.mocked(api.post).mockResolvedValue({ data: { createdCount: 1, deletedCount: 1 } } as never)

    await DirectoryService.bulkDeleteDirectories(7, [1, 2])
    await DirectoryService.bulkCreateDirectories(7, ["https://a.example.com/admin"])
    await DirectoryService.exportDirectoriesByTargetId(7)
    await DirectoryService.exportDirectoriesByScanId(12)

    expect(api.post).toHaveBeenNthCalledWith(1, "/directories:batchDelete", { names: ["targets/7/directories/1", "targets/7/directories/2"] })
    expect(api.post).toHaveBeenNthCalledWith(2, "/targets/7/directories:batchCreate", { urls: ["https://a.example.com/admin"] })
    expect(api.get).toHaveBeenNthCalledWith(1, "/targets/7/directories/exportFiles/current", { responseType: "blob" })
    expect(api.get).toHaveBeenNthCalledWith(2, "/scans/12/directories/exportFiles/current", { responseType: "blob" })
  })
})
