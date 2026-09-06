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
import { OrganizationService } from "@/services/organization.service"

describe("organization.service contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("lists organizations through the backend collection path and adapts AIP names for the UI", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [
          {
            id: 7,
            name: "organizations/7",
            displayName: "Acme",
            description: "Platform team",
            createdAt: "2026-06-14T01:02:03Z",
            targetCount: 3,
          },
        ],
        totalSize: 1,
      },
    } as never)

    const result = await OrganizationService.getOrganizations({
      pageSize: 25,
      pageToken: "token-2",
      filter: 'displayName="platform"',
      orderBy: "createdAt desc",
    })

    expect(api.get).toHaveBeenCalledWith("/organizations", {
      params: {
        pageSize: 25,
        pageToken: "token-2",
        filter: 'displayName="platform"',
        orderBy: "createdAt desc",
      },
    })
    expect(result).toEqual({
      results: [
        {
          id: 7,
          name: "Acme",
          resourceName: "organizations/7",
          description: "Platform team",
          createdAt: "2026-06-14T01:02:03Z",
          updatedAt: "2026-06-14T01:02:03Z",
          targetCount: 3,
        },
      ],
      total: 1,
      totalSize: 1,
      nextPageToken: undefined,
      page: 1,
      pageSize: 25,
      totalPages: 1,
    })
  })

  it("does not send legacy page or sorting aliases for organization collection queries", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [],
        totalSize: 0,
        nextPageToken: "",
      },
    } as never)

    await OrganizationService.getOrganizations({
      pageSize: 10,
      filter: 'displayName="acme"',
      orderBy: "displayName",
    })

    const [, options] = vi.mocked(api.get).mock.calls.at(-1) ?? []
    expect(options).toEqual({
      params: {
        pageSize: 10,
        filter: 'displayName="acme"',
        orderBy: "displayName",
      },
    })
    expect(options?.params).not.toHaveProperty("page")
    expect(options?.params).not.toHaveProperty("sort")
    expect(options?.params).not.toHaveProperty("sortBy")
    expect(options?.params).not.toHaveProperty("sortOrder")
    expect(options?.params).not.toHaveProperty("keyword")
  })

  it("updates organizations with PATCH updateMask payloads", async () => {
    vi.mocked(api.patch).mockResolvedValue({
      data: {
        id: 7,
        name: "organizations/7",
        displayName: "Acme Labs",
        description: "Updated",
        createdAt: "2026-06-14T01:02:03Z",
        targetCount: 3,
      },
    } as never)

    const result = await OrganizationService.updateOrganization({
      id: 7,
      name: "Acme Labs",
      description: "Updated",
    })

    expect(api.patch).toHaveBeenCalledWith("/organizations/7", {
      name: "organizations/7",
      displayName: "Acme Labs",
      description: "Updated",
      updateMask: "displayName,description",
    })
    expect(result.name).toBe("Acme Labs")
    expect(result.resourceName).toBe("organizations/7")
  })

  it("normalizes no-content organization deletion for optimistic UI hooks", async () => {
    vi.mocked(api.delete).mockResolvedValue({ status: 204, data: undefined } as never)

    await expect(OrganizationService.deleteOrganization(7)).resolves.toEqual({
      id: 7,
      organizationName: "organizations/7",
      deletedCount: 1,
    })

    expect(api.delete).toHaveBeenCalledWith("/organizations/7")
  })

  it("uses resource names for batch delete and normalizes deletedCount", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { deletedCount: 2 } } as never)

    const result = await OrganizationService.batchDeleteOrganizations([7, 8])

    expect(api.post).toHaveBeenCalledWith("/organizations:batchDelete", {
      names: ["organizations/7", "organizations/8"],
    })
    expect(result).toEqual({
      deletedCount: 2,
    })
  })

  it("lists organization targets with the backend cursor token and adapts target display names", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [
          {
            id: 11,
            name: "targets/11",
            displayName: "api.example.com",
            type: "domain",
            createdAt: "2026-06-14T01:02:03Z",
          },
        ],
        totalSize: 1,
        nextPageToken: "organization-targets-page-two",
      },
    } as never)

    const result = await OrganizationService.getOrganizationTargets(7, {
      pageSize: 10,
      pageToken: "organization-targets-page-one",
      search: "api",
      type: "domain",
    })

    expect(api.get).toHaveBeenCalledWith("/organizations/7/targets", {
      params: {
        pageSize: 10,
        pageToken: "organization-targets-page-one",
        filter: "api",
        type: "domain",
      },
    })
    const [, options] = vi.mocked(api.get).mock.calls.at(-1) ?? []
    expect(options?.params).not.toHaveProperty("page")
    expect(result.results).toEqual([
      {
        id: 11,
        name: "api.example.com",
        resourceName: "targets/11",
        type: "domain",
        createdAt: "2026-06-14T01:02:03Z",
      },
    ])
    expect(result.nextPageToken).toBe("organization-targets-page-two")
  })

  it("passes explicit smart filter syntax to organization target queries", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [],
        totalSize: 0,
      },
    } as never)

    await OrganizationService.getOrganizationTargets(7, {
      pageSize: 10,
      pageToken: "organization-targets-page-one",
      filter: 'type="domain" || type="ip"',
      search: "ignored when filter is explicit",
    })

    expect(api.get).toHaveBeenCalledWith("/organizations/7/targets", {
      params: {
        pageSize: 10,
        pageToken: "organization-targets-page-one",
        filter: 'type="domain" || type="ip"',
      },
    })
  })
})
