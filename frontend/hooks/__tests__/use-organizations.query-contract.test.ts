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

  it("分页元信息只依赖主线 camelCase 字段", async () => {
    organizationServiceMocks.getOrganizations.mockResolvedValue({
      results: [{ id: 1, name: "org-1", description: "", createdAt: "2026-01-01T00:00:00Z", updatedAt: "2026-01-01T00:00:00Z" }],
      total: 11,
      count: 99,
      page: 2,
      pageSize: 5,
      totalPages: 3,
    })

    const { result } = renderHookWithProviders(() =>
      useOrganizations({ page: 2, pageSize: 20, filter: "demo" })
    )

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(organizationServiceMocks.getOrganizations).toHaveBeenCalledWith({
      page: 2,
      pageSize: 20,
      filter: "demo",
    })
    expect(result.current.data).toEqual({
      organizations: [
        {
          id: 1,
          name: "org-1",
          description: "",
          createdAt: "2026-01-01T00:00:00Z",
          updatedAt: "2026-01-01T00:00:00Z",
        },
      ],
      pagination: {
        total: 11,
        page: 2,
        pageSize: 5,
        totalPages: 3,
      },
    })
  })
})
