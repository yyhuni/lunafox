import { waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { useOrganizations } from "@/hooks/use-organizations"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"

const organizationServiceMocks = vi.hoisted(() => ({
  getOrganizations: vi.fn(),
  getOrganizationById: vi.fn(),
  getOrganizationTargets: vi.fn(),
  createOrganization: vi.fn(),
  updateOrganization: vi.fn(),
  deleteOrganization: vi.fn(),
  batchDeleteOrganizations: vi.fn(),
  linkTarget: vi.fn(),
  unlinkTarget: vi.fn(),
  batchLinkTargets: vi.fn(),
  batchUnlinkTargets: vi.fn(),
}))

vi.mock("@/services/organization.service", () => ({
  OrganizationService: organizationServiceMocks,
}))

describe("use-organizations query contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("uses canonical backend list query controls and exposes pagination tokens", async () => {
    organizationServiceMocks.getOrganizations.mockResolvedValue({
      results: [{ id: 1, name: "org-1", description: "", createdAt: "2026-01-01T00:00:00Z", updatedAt: "2026-01-01T00:00:00Z" }],
      totalSize: 11,
      nextPageToken: "token-3",
      pageSize: 5,
      totalPages: 3,
    })

    const { result } = renderHookWithProviders(() =>
      useOrganizations({
        pageSize: 20,
        pageToken: "token-2",
        filter: 'displayName="demo"',
        orderBy: "displayName",
      })
    )

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(organizationServiceMocks.getOrganizations).toHaveBeenCalledWith({
      pageSize: 20,
      pageToken: "token-2",
      filter: 'displayName="demo"',
      orderBy: "displayName",
    })
    expect(result.current.data).toEqual({
      results: [
        {
          id: 1,
          name: "org-1",
          description: "",
          createdAt: "2026-01-01T00:00:00Z",
          updatedAt: "2026-01-01T00:00:00Z",
        },
      ],
      organizations: [
        {
          id: 1,
          name: "org-1",
          description: "",
          createdAt: "2026-01-01T00:00:00Z",
          updatedAt: "2026-01-01T00:00:00Z",
        },
      ],
      total: 11,
      totalSize: 11,
      nextPageToken: "token-3",
      page: 1,
      pageSize: 5,
      totalPages: 3,
      pagination: {
        total: 11,
        page: 1,
        pageSize: 5,
        totalPages: 3,
      },
    })
  })
})
