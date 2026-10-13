import { beforeEach, describe, expect, it, vi } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const apiMocks = vi.hoisted(() => ({ post: vi.fn() }))

vi.mock("@/lib/api-client", () => ({ api: apiMocks }))

import { api } from "@/lib/api-client"
import { SearchExportService } from "@/services/search-export.service"

const source = readFileSync(path.resolve(process.cwd(), "services/search-export.service.ts"), "utf8")

describe("search-export.service contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    document.body.innerHTML = ""
  })

  it("keeps full export out of Blob aggregation and pagination", () => {
    expect(source).toContain('api.post<GlobalAssetExportGrantResponse>("/assets:export"')
    expect(source).toContain("startNativeDownload")
    expect(source).not.toContain("responseType: \"blob\"")
    expect(source).not.toContain("pageSize")
    expect(source).not.toContain("pageToken")
  })

  it("exchanges the submitted query for a one-time native download URL", async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: {
        downloadUrl: "/v1/assets/exportFiles/current?downloadToken=opaque-token",
        expiresAt: "2026-10-13T00:02:00Z",
      },
    } as never)
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => undefined)

    await SearchExportService.exportAll({ q: 'host="api"', assetType: "endpoint" })

    expect(api.post).toHaveBeenCalledWith("/assets:export", {
      q: 'host="api"',
      assetType: "endpoint",
    })
    expect(click).toHaveBeenCalledOnce()
    expect(document.querySelector("a")).toBeNull()
    click.mockRestore()
  })

  it("does not expose the primary JWT or page parameters to the download URL", async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: {
        downloadUrl: "/v1/assets/exportFiles/current?downloadToken=opaque-token",
        expiresAt: "2026-10-13T00:02:00Z",
      },
    } as never)
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => undefined)

    await SearchExportService.exportAll({ q: 'host="api"', assetType: "website" })

    const anchor = click.mock.instances[0] as HTMLAnchorElement
    const url = new URL(anchor.href)
    expect(anchor.href).toContain("downloadToken=opaque-token")
    expect(url.searchParams.has("q")).toBe(false)
    expect(url.searchParams.has("assetType")).toBe(false)
    expect(url.searchParams.has("pageSize")).toBe(false)
    expect(url.searchParams.has("Authorization")).toBe(false)
    click.mockRestore()
  })

  it("rejects missing grant URLs instead of starting a browser request", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { expiresAt: "2026-10-13T00:02:00Z" } } as never)

    await expect(SearchExportService.exportAll({ q: "api", assetType: "website" }))
      .rejects.toThrow("download URL")
    expect(document.querySelector("a")).toBeNull()
  })

  it("rejects malformed grant expiry metadata", async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: {
        downloadUrl: "/v1/assets/exportFiles/current?downloadToken=opaque-token",
        expiresAt: "not-a-timestamp",
      },
    } as never)

    await expect(SearchExportService.exportAll({ q: "api", assetType: "website" }))
      .rejects.toThrow("valid expiry")
    expect(document.querySelector("a")).toBeNull()
  })
})
