import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  subdomainKeys,
  useBatchDeleteSubdomains,
  useBulkCreateSubdomains,
  useDeleteSubdomain,
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

describe("use-subdomains mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("批量创建子域名含跳过项时走 partialSuccess 并保持失效范围", async () => {
    subdomainServiceMocks.bulkCreateSubdomains.mockResolvedValue({
      message: "ok",
      createdCount: 3,
      skippedCount: 1,
      invalidCount: 1,
      mismatchedCount: 0,
      totalReceived: 5,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useBulkCreateSubdomains(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        targetId: 7,
        subdomains: ["a.example.com", "b.example.com", "c.example.com"],
      })
    })

    expect(subdomainServiceMocks.bulkCreateSubdomains).toHaveBeenCalledWith(7, [
      "a.example.com",
      "b.example.com",
      "c.example.com",
    ])
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.batchCreating",
      {},
      "bulk-create-subdomains"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("bulk-create-subdomains")
    expect(toastMocks.warning).toHaveBeenCalledWith(
      "toast.asset.subdomain.create.partialSuccess",
      { success: 3, skipped: 2 }
    )
    expect(invalidateSpy).toHaveBeenNthCalledWith(1, {
      queryKey: ["targets", 7, "subdomains"],
      exact: false,
      refetchType: "active",
    })
    expect(invalidateSpy).toHaveBeenNthCalledWith(2, {
      queryKey: subdomainKeys.all,
      exact: false,
      refetchType: "active",
    })
  })

  it("删除子域名失败时保留错误码映射与 fallback key", async () => {
    subdomainServiceMocks.deleteSubdomain.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useDeleteSubdomain())

    await act(async () => {
      await expect(result.current.mutateAsync(12)).rejects.toBeDefined()
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.deleting",
      {},
      "delete-subdomain-12"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("delete-subdomain-12")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.asset.subdomain.delete.error"
    )
  })

  it("删除子域名成功时保留 success 提示与失效范围", async () => {
    subdomainServiceMocks.deleteSubdomain.mockResolvedValue(undefined)

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useDeleteSubdomain(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(12)
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.deleting",
      {},
      "delete-subdomain-12"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("delete-subdomain-12")
    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.asset.subdomain.delete.success"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: subdomainKeys.all })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["targets"] })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["scans"] })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["organizations"] })
  })

  it("批量删除子域名成功时保留 bulkSuccess 提示与失效范围", async () => {
    subdomainServiceMocks.batchDeleteSubdomains.mockResolvedValue({
      message: "ok",
      deletedCount: 2,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useBatchDeleteSubdomains(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync([1, 2])
    })

    expect(subdomainServiceMocks.batchDeleteSubdomains).toHaveBeenCalledWith([1, 2])
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.batchDeleting",
      {},
      "batch-delete-subdomains"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("batch-delete-subdomains")
    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.asset.subdomain.delete.bulkSuccess",
      { count: 2 }
    )
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: subdomainKeys.all })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["targets"] })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["scans"] })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["organizations"] })
  })
})
