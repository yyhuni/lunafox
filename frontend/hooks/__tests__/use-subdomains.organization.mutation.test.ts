import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  subdomainKeys,
  useBatchDeleteSubdomainsFromOrganization,
  useDeleteSubdomainFromOrganization,
} from "@/hooks/use-subdomains"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const subdomainServiceMocks = vi.hoisted(() => ({
  bulkCreateSubdomains: vi.fn(),
  createSubdomains: vi.fn(),
  getSubdomainById: vi.fn(),
  updateSubdomain: vi.fn(),
  bulkDeleteSubdomains: vi.fn(),
  deleteSubdomain: vi.fn(),
  batchDeleteSubdomains: vi.fn(),
  batchDeleteSubdomainsFromOrganization: vi.fn(),
  getSubdomainsByOrgId: vi.fn(),
  getAllSubdomains: vi.fn(),
  getSubdomainsByTargetId: vi.fn(),
  getSubdomainsByScanId: vi.fn(),
  exportSubdomainsByTargetId: vi.fn(),
  exportSubdomainsByScanId: vi.fn(),
}))

const organizationServiceMocks = vi.hoisted(() => ({
  getOrganizations: vi.fn(),
  getOrganizationById: vi.fn(),
  getOrganizationTargets: vi.fn(),
  createOrganization: vi.fn(),
  updateOrganization: vi.fn(),
  deleteOrganization: vi.fn(),
  batchDeleteOrganizations: vi.fn(),
  linkTargetToOrganization: vi.fn(),
  unlinkTargetsFromOrganization: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/subdomain.service", () => ({
  SubdomainService: subdomainServiceMocks,
}))

vi.mock("@/services/organization.service", () => ({
  OrganizationService: organizationServiceMocks,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

describe("use-subdomains organization mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("从组织移除单个子域名成功时保持成功提示与失效范围", async () => {
    organizationServiceMocks.unlinkTargetsFromOrganization.mockResolvedValue({
      unlinkedCount: 1,
      message: "ok",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useDeleteSubdomainFromOrganization(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        organizationId: 5,
        targetId: 42,
      })
    })

    expect(organizationServiceMocks.unlinkTargetsFromOrganization).toHaveBeenCalledWith({
      organizationId: 5,
      targetIds: [42],
    })
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.removing",
      {},
      "delete-5-42"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("delete-5-42")
    expect(toastMocks.success).toHaveBeenCalledWith("toast.asset.subdomain.delete.success")
    expect(invalidateSpy).toHaveBeenNthCalledWith(1, { queryKey: subdomainKeys.all })
    expect(invalidateSpy).toHaveBeenNthCalledWith(2, { queryKey: ["organizations"] })
  })

  it("批量从组织移除子域名成功时保留 bulkSuccess 提示与失效范围", async () => {
    subdomainServiceMocks.batchDeleteSubdomainsFromOrganization.mockResolvedValue({
      message: "ok",
      successCount: 3,
      failedCount: 0,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useBatchDeleteSubdomainsFromOrganization(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        organizationId: 5,
        domainIds: [101, 102, 103],
      })
    })

    expect(subdomainServiceMocks.batchDeleteSubdomainsFromOrganization).toHaveBeenCalledWith({
      organizationId: 5,
      domainIds: [101, 102, 103],
    })
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.batchRemoving",
      {},
      "batch-delete-5"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("batch-delete-5")
    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.asset.subdomain.delete.bulkSuccess",
      { count: 3 }
    )
    expect(invalidateSpy).toHaveBeenNthCalledWith(1, { queryKey: subdomainKeys.all })
    expect(invalidateSpy).toHaveBeenNthCalledWith(2, { queryKey: ["organizations"] })
  })

  it("从组织移除子域名失败时保留错误码映射与 fallback key", async () => {
    organizationServiceMocks.unlinkTargetsFromOrganization.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useDeleteSubdomainFromOrganization())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          organizationId: 5,
          targetId: 42,
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.removing",
      {},
      "delete-5-42"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("delete-5-42")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.asset.subdomain.delete.error"
    )
  })

  it("批量从组织移除子域名失败时保留错误码映射与 fallback key", async () => {
    subdomainServiceMocks.batchDeleteSubdomainsFromOrganization.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useBatchDeleteSubdomainsFromOrganization())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          organizationId: 5,
          domainIds: [101],
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.batchRemoving",
      {},
      "batch-delete-5"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("batch-delete-5")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.asset.subdomain.delete.error"
    )
  })
})
