import { beforeEach, describe, expect, it, vi } from "vitest"

const apiMocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
}))

vi.mock("@/lib/api-client", () => ({
  api: apiMocks,
}))

import { api } from "@/lib/api-client"
import {
  batchCreateTargets,
  batchDeleteTargets,
  getTargets,
  updateTarget,
} from "@/services/target.service"

describe("target.service contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("lists targets through the AIP collection path and adapts canonical list controls", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [
          {
            id: 7,
            name: "targets/7",
            displayName: "example.com",
            type: "domain",
            createdAt: "2026-06-14T01:02:03Z",
            organizations: [
              {
                id: 3,
                name: "organizations/3",
                displayName: "Acme",
              },
            ],
          },
        ],
        totalSize: 42,
        nextPageToken: "next-token",
      },
    } as never)

    const result = await getTargets({
      pageSize: 25,
      pageToken: "page-token-2",
      filter: 'displayName="demo" && (type=="domain" || type=="ip")',
      orderBy: "createdAt desc",
    })

    expect(api.get).toHaveBeenCalledWith("/targets", {
      params: {
        pageSize: 25,
        pageToken: "page-token-2",
        filter: 'displayName="demo" && (type=="domain" || type=="ip")',
        orderBy: "createdAt desc",
      },
    })
    expect(result.results).toEqual([
      {
        id: 7,
        name: "example.com",
        resourceName: "targets/7",
        type: "domain",
        createdAt: "2026-06-14T01:02:03Z",
        organizations: [
          {
            id: 3,
            name: "Acme",
            resourceName: "organizations/3",
          },
        ],
      },
    ])
    expect(result.total).toBe(42)
    expect(result.totalSize).toBe(42)
    expect(result.nextPageToken).toBe("next-token")
  })

  it("creates targets with AIP organization resource names", async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: {
        createdCount: 1,
        failedCount: 0,
        failedTargets: [],
        message: "ok",
      },
    } as never)

    await batchCreateTargets({
      targets: [{ name: "example.com" }],
      organizationIds: [3],
    })

    expect(api.post).toHaveBeenCalledWith("/targets:batchCreate", {
      targets: [{ name: "example.com" }],
      organization: "organizations/3",
    })
  })

  it("updates targets with AIP updateMask payloads", async () => {
    vi.mocked(api.patch).mockResolvedValue({
      data: {
        id: 7,
        name: "targets/7",
        displayName: "renamed.example.com",
        type: "domain",
        createdAt: "2026-06-14T01:02:03Z",
      },
    } as never)

    await updateTarget(7, { name: "renamed.example.com" })

    expect(api.patch).toHaveBeenCalledWith("/targets/7", {
      name: "targets/7",
      displayName: "renamed.example.com",
      updateMask: "displayName",
    })
  })

  it("deletes targets with AIP resource names for batch operations", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { deletedCount: 2 } } as never)

    await batchDeleteTargets({ ids: [7, 8] })

    expect(api.post).toHaveBeenCalledWith("/targets:batchDelete", {
      names: ["targets/7", "targets/8"],
    })
  })
})
