import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  targetKeys,
  useBatchCreateTargets,
  useBatchDeleteTargets,
  useCreateTarget,
  useDeleteTarget,
  useLinkTargetOrganizations,
  useUnlinkTargetOrganizations,
  useUpdateTarget,
  useUpdateTargetBlacklist,
} from "@/hooks/use-targets"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const targetServiceMocks = vi.hoisted(() => ({
  getTargets: vi.fn(),
  getTargetById: vi.fn(),
  createTarget: vi.fn(),
  updateTarget: vi.fn(),
  deleteTarget: vi.fn(),
  batchDeleteTargets: vi.fn(),
  batchCreateTargets: vi.fn(),
  getTargetOrganizations: vi.fn(),
  linkTargetOrganizations: vi.fn(),
  unlinkTargetOrganizations: vi.fn(),
  getTargetEndpoints: vi.fn(),
  getTargetBlacklist: vi.fn(),
  updateTargetBlacklist: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/target.service", () => ({
  getTargets: targetServiceMocks.getTargets,
  getTargetById: targetServiceMocks.getTargetById,
  createTarget: targetServiceMocks.createTarget,
  updateTarget: targetServiceMocks.updateTarget,
  deleteTarget: targetServiceMocks.deleteTarget,
  batchDeleteTargets: targetServiceMocks.batchDeleteTargets,
  batchCreateTargets: targetServiceMocks.batchCreateTargets,
  getTargetOrganizations: targetServiceMocks.getTargetOrganizations,
  linkTargetOrganizations: targetServiceMocks.linkTargetOrganizations,
  unlinkTargetOrganizations: targetServiceMocks.unlinkTargetOrganizations,
  getTargetEndpoints: targetServiceMocks.getTargetEndpoints,
  getTargetBlacklist: targetServiceMocks.getTargetBlacklist,
  updateTargetBlacklist: targetServiceMocks.updateTargetBlacklist,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

describe("use-targets mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("删除目标时保持 loading/dismiss/success 与 query 失效行为", async () => {
    targetServiceMocks.deleteTarget.mockResolvedValue(undefined)
    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")

    const { result } = renderHookWithProviders(() => useDeleteTarget(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({ id: 12, name: "example.com" })
    })

    expect(targetServiceMocks.deleteTarget).toHaveBeenCalledWith(12)
    expect(toastMocks.loading).toHaveBeenCalledWith("common.status.deleting", {}, "delete-target-12")
    expect(toastMocks.dismiss).toHaveBeenCalledWith("delete-target-12")
    expect(toastMocks.success).toHaveBeenCalledWith("toast.target.delete.success", { name: "example.com" })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: targetKeys.all })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["organizations"] })
  })

  it("创建目标成功时保留 success 提示与列表失效范围", async () => {
    targetServiceMocks.createTarget.mockResolvedValue({
      id: 1,
      name: "example.com",
      createdAt: "2026-02-11T00:00:00Z",
      updatedAt: "2026-02-11T00:00:00Z",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useCreateTarget(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({ name: "example.com" })
    })

    expect(targetServiceMocks.createTarget).toHaveBeenCalledWith({ name: "example.com" })
    expect(toastMocks.success).toHaveBeenCalledWith("toast.target.create.success")
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: targetKeys.all })
  })

  it("创建目标失败时保持错误码映射与回退 key", async () => {
    targetServiceMocks.createTarget.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "CONFLICT",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useCreateTarget())

    await act(async () => {
      await expect(result.current.mutateAsync({ name: "conflict-target" })).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith("CONFLICT", "toast.target.create.error")
  })

  it("更新目标成功时失效列表与详情缓存", async () => {
    targetServiceMocks.updateTarget.mockResolvedValue({
      id: 7,
      name: "updated.com",
      updatedAt: "2026-02-11T00:00:00Z",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUpdateTarget(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        id: 7,
        data: {
          name: "updated.com",
        },
      })
    })

    expect(targetServiceMocks.updateTarget).toHaveBeenCalledWith(7, { name: "updated.com" })
    expect(toastMocks.success).toHaveBeenCalledWith("toast.target.update.success")
    expect(invalidateSpy).toHaveBeenNthCalledWith(1, { queryKey: targetKeys.all })
    expect(invalidateSpy).toHaveBeenNthCalledWith(2, { queryKey: targetKeys.detail(7) })
  })

  it("更新目标失败时保持错误码映射与回退 key", async () => {
    targetServiceMocks.updateTarget.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useUpdateTarget())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          id: 7,
          data: {
            name: "forbidden.com",
          },
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith("FORBIDDEN", "toast.target.update.error")
  })

  it("删除目标失败时保留 loading/dismiss 与错误码映射", async () => {
    targetServiceMocks.deleteTarget.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useDeleteTarget())

    await act(async () => {
      await expect(result.current.mutateAsync({ id: 12, name: "blocked.com" })).rejects.toBeDefined()
    })

    expect(toastMocks.loading).toHaveBeenCalledWith("common.status.deleting", {}, "delete-target-12")
    expect(toastMocks.dismiss).toHaveBeenCalledWith("delete-target-12")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith("FORBIDDEN", "toast.target.delete.error")
  })

  it("批量删除目标成功时提示数量并失效列表", async () => {
    targetServiceMocks.batchDeleteTargets.mockResolvedValue({
      deletedCount: 3,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useBatchDeleteTargets(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({ ids: [1, 2, 3] })
    })

    expect(targetServiceMocks.batchDeleteTargets).toHaveBeenCalledWith({ ids: [1, 2, 3] })
    expect(toastMocks.success).toHaveBeenCalledWith("toast.target.delete.bulkSuccess", { count: 3 })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: targetKeys.all })
  })

  it("批量删除目标失败时保留错误码映射与回退 key", async () => {
    targetServiceMocks.batchDeleteTargets.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useBatchDeleteTargets())

    await act(async () => {
      await expect(result.current.mutateAsync({ ids: [4, 5] })).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith("FORBIDDEN", "toast.target.delete.error")
  })

  it("批量创建目标成功时提示数量并失效列表与组织", async () => {
    targetServiceMocks.batchCreateTargets.mockResolvedValue({
      createdCount: 2,
      reusedCount: 0,
      failedCount: 0,
      failedTargets: [],
      message: "ok",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useBatchCreateTargets(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        targets: [{ name: "alpha.com" }, { name: "beta.com" }],
      })
    })

    expect(targetServiceMocks.batchCreateTargets).toHaveBeenCalledWith({
      targets: [{ name: "alpha.com" }, { name: "beta.com" }],
    })
    expect(toastMocks.success).toHaveBeenCalledWith("toast.target.create.bulkSuccess", { count: 2 })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: targetKeys.all })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["organizations"] })
  })

  it("批量创建目标失败时保留错误码映射与回退 key", async () => {
    targetServiceMocks.batchCreateTargets.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "CONFLICT",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useBatchCreateTargets())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          targets: [{ name: "conflict.com" }],
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith("CONFLICT", "toast.target.create.error")
  })

  it("关联组织成功时失效组织列表与目标详情", async () => {
    targetServiceMocks.linkTargetOrganizations.mockResolvedValue({ message: "ok" })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useLinkTargetOrganizations(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({ targetId: 5, organizationIds: [1, 2] })
    })

    expect(targetServiceMocks.linkTargetOrganizations).toHaveBeenCalledWith(5, [1, 2])
    expect(toastMocks.success).toHaveBeenCalledWith("toast.target.link.success")
    expect(invalidateSpy).toHaveBeenNthCalledWith(1, {
      queryKey: targetKeys.organizations(5, 1, 10),
      exact: false,
    })
    expect(invalidateSpy).toHaveBeenNthCalledWith(2, {
      queryKey: targetKeys.detail(5),
    })
  })

  it("关联组织失败时保留错误码映射与回退 key", async () => {
    targetServiceMocks.linkTargetOrganizations.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useLinkTargetOrganizations())

    await act(async () => {
      await expect(
        result.current.mutateAsync({ targetId: 5, organizationIds: [1, 2] })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith("FORBIDDEN", "toast.target.link.error")
  })

  it("取消关联成功时失效组织列表与目标详情", async () => {
    targetServiceMocks.unlinkTargetOrganizations.mockResolvedValue({ message: "ok" })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUnlinkTargetOrganizations(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({ targetId: 5, organizationIds: [3] })
    })

    expect(targetServiceMocks.unlinkTargetOrganizations).toHaveBeenCalledWith(5, [3])
    expect(toastMocks.success).toHaveBeenCalledWith("toast.target.unlink.success")
    expect(invalidateSpy).toHaveBeenNthCalledWith(1, {
      queryKey: targetKeys.organizations(5, 1, 10),
      exact: false,
    })
    expect(invalidateSpy).toHaveBeenNthCalledWith(2, {
      queryKey: targetKeys.detail(5),
    })
  })

  it("取消关联失败时保留错误码映射与回退 key", async () => {
    targetServiceMocks.unlinkTargetOrganizations.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useUnlinkTargetOrganizations())

    await act(async () => {
      await expect(
        result.current.mutateAsync({ targetId: 5, organizationIds: [3] })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith("FORBIDDEN", "toast.target.unlink.error")
  })

  it("更新黑名单成功时失效 blacklist 缓存并提示成功", async () => {
    targetServiceMocks.updateTargetBlacklist.mockResolvedValue({
      patterns: ["admin"],
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUpdateTargetBlacklist(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({ targetId: 9, patterns: ["admin"] })
    })

    expect(targetServiceMocks.updateTargetBlacklist).toHaveBeenCalledWith(9, ["admin"])
    expect(toastMocks.success).toHaveBeenCalledWith("toast.blacklist.save.success")
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: targetKeys.blacklist(9) })
  })

  it("更新黑名单失败时保留错误码映射与回退 key", async () => {
    targetServiceMocks.updateTargetBlacklist.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useUpdateTargetBlacklist())

    await act(async () => {
      await expect(
        result.current.mutateAsync({ targetId: 9, patterns: ["admin"] })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith("FORBIDDEN", "toast.blacklist.save.error")
  })
})
