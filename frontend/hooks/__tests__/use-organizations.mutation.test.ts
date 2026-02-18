import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  organizationKeys,
  useBatchDeleteOrganizations,
  useCreateOrganization,
  useDeleteOrganization,
  useUpdateOrganization,
} from "@/hooks/use-organizations"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

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

vi.mock("@/services/organization.service", () => ({
  OrganizationService: organizationServiceMocks,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

describe("use-organizations mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("创建组织成功时保持 loading/dismiss/success 和组织列表失效", async () => {
    organizationServiceMocks.createOrganization.mockResolvedValue({
      id: 1,
      name: "acme",
      description: "Acme Org",
      createdAt: "2026-02-11T00:00:00Z",
      updatedAt: "2026-02-11T00:00:00Z",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useCreateOrganization(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        name: "acme",
        description: "Acme Org",
      })
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.creating",
      {},
      "create-organization"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("create-organization")
    expect(toastMocks.success).toHaveBeenCalledWith("toast.organization.create.success")
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: organizationKeys.all,
    })
  })

  it("创建组织失败时保持 loading/dismiss 与错误码映射", async () => {
    organizationServiceMocks.createOrganization.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "CONFLICT",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useCreateOrganization())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          name: "conflict-org",
          description: "Conflict",
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.creating",
      {},
      "create-organization"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("create-organization")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "CONFLICT",
      "toast.organization.create.error"
    )
  })

  it("更新组织成功时保持 loading/dismiss/success 和组织列表失效", async () => {
    organizationServiceMocks.updateOrganization.mockResolvedValue({
      id: 4,
      name: "acme-updated",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUpdateOrganization(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        id: 4,
        data: {
          name: "acme-updated",
          description: "Updated",
        },
      })
    })

    expect(organizationServiceMocks.updateOrganization).toHaveBeenCalledWith({
      id: 4,
      name: "acme-updated",
      description: "Updated",
    })
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.updating",
      {},
      "update-4"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("update-4")
    expect(toastMocks.success).toHaveBeenCalledWith("toast.organization.update.success")
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: organizationKeys.all,
    })
  })

  it("更新组织失败时保持 loading/dismiss 与错误码映射", async () => {
    organizationServiceMocks.updateOrganization.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useUpdateOrganization())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          id: 4,
          data: {
            name: "forbidden-org",
            description: "Forbidden",
          },
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.updating",
      {},
      "update-4"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("update-4")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.organization.update.error"
    )
  })

  it("删除组织成功时触发删除提示并在 settled 后失效 organizations/targets", async () => {
    organizationServiceMocks.deleteOrganization.mockResolvedValue({
      message: "deleted",
      organizationId: 9,
      organizationName: "legacy-org",
      deletedCount: 1,
      deletedOrganizations: ["legacy-org"],
      detail: {
        phase1: "ok",
        phase2: "ok",
      },
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useDeleteOrganization(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(9)
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.deleting",
      {},
      "delete-9"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("delete-9")
    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.organization.delete.success",
      { name: "legacy-org" }
    )
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["organizations"],
    })
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["targets"],
    })
  })

  it("删除组织失败时回滚并触发错误码提示", async () => {
    organizationServiceMocks.deleteOrganization.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useDeleteOrganization(), {
      queryClient,
    })

    await act(async () => {
      await expect(result.current.mutateAsync(9)).rejects.toBeDefined()
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.deleting",
      {},
      "delete-9"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("delete-9")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.organization.delete.error"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["organizations"],
    })
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["targets"],
    })
  })

  it("批量删除组织成功时提示数量并在 settled 后失效 organizations/targets", async () => {
    organizationServiceMocks.batchDeleteOrganizations.mockResolvedValue({
      deletedCount: 2,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useBatchDeleteOrganizations(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync([3, 4])
    })

    expect(organizationServiceMocks.batchDeleteOrganizations).toHaveBeenCalledWith([3, 4])
    expect(toastMocks.dismiss).toHaveBeenCalledWith("batch-delete")
    expect(toastMocks.success).toHaveBeenCalledWith("toast.organization.delete.bulkSuccess", {
      count: 2,
    })
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["organizations"],
    })
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["targets"],
    })
  })

  it("批量删除组织失败时回滚并触发错误码提示", async () => {
    organizationServiceMocks.batchDeleteOrganizations.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useBatchDeleteOrganizations(), {
      queryClient,
    })

    await act(async () => {
      await expect(result.current.mutateAsync([3, 4])).rejects.toBeDefined()
    })

    expect(toastMocks.dismiss).toHaveBeenCalledWith("batch-delete")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.organization.delete.error"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["organizations"],
    })
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["targets"],
    })
  })
})
