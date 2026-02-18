import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  useBulkDeleteWebSites,
  useBulkCreateWebsites,
  useDeleteWebSite,
  websiteKeys,
} from "@/hooks/use-websites"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const websiteServiceMocks = vi.hoisted(() => ({
  getTargetWebSites: vi.fn(),
  getScanWebSites: vi.fn(),
  deleteWebSite: vi.fn(),
  bulkDeleteWebSites: vi.fn(),
  bulkDelete: vi.fn(),
  bulkCreateWebsites: vi.fn(),
  exportWebsitesByTargetId: vi.fn(),
  exportWebsitesByScanId: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/website.service", () => ({
  WebsiteService: websiteServiceMocks,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

describe("use-websites mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("批量创建网站成功时保留成功提示与失效范围", async () => {
    websiteServiceMocks.bulkCreateWebsites.mockResolvedValue({
      message: "ok",
      createdCount: 2,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useBulkCreateWebsites(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        targetId: 7,
        urls: ["https://a.example.com", "https://b.example.com"],
      })
    })

    expect(websiteServiceMocks.bulkCreateWebsites).toHaveBeenCalledWith(7, [
      "https://a.example.com",
      "https://b.example.com",
    ])
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.batchCreating",
      {},
      "bulk-create-websites"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("bulk-create-websites")
    expect(toastMocks.success).toHaveBeenCalledWith("toast.asset.website.create.success", {
      count: 2,
    })
    expect(invalidateSpy).toHaveBeenNthCalledWith(1, {
      queryKey: websiteKeys.all,
      exact: false,
      refetchType: "active",
    })
    expect(invalidateSpy).toHaveBeenNthCalledWith(2, {
      queryKey: ["targets", 7],
      refetchType: "active",
    })
  })

  it("批量创建网站 createdCount 为 0 时走 partialSuccess 警告路径", async () => {
    websiteServiceMocks.bulkCreateWebsites.mockResolvedValue({
      message: "ok",
      createdCount: 0,
    })

    const { result } = renderHookWithProviders(() => useBulkCreateWebsites())

    await act(async () => {
      await result.current.mutateAsync({
        targetId: 7,
        urls: ["https://a.example.com"],
      })
    })

    expect(toastMocks.warning).toHaveBeenCalledWith(
      "toast.asset.website.create.partialSuccess",
      { success: 0, skipped: 0 }
    )
    expect(toastMocks.success).not.toHaveBeenCalled()
  })

  it("批量删除网站成功时保留 bulkSuccess 提示与失效范围", async () => {
    websiteServiceMocks.bulkDeleteWebSites.mockResolvedValue({
      message: "ok",
      deletedCount: 2,
      requestedIds: [11, 12],
      cascadeDeleted: {},
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useBulkDeleteWebSites(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync([11, 12])
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.batchDeleting",
      {},
      "bulk-delete-websites"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("bulk-delete-websites")
    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.asset.website.delete.bulkSuccess",
      { count: 2 }
    )
    expect(invalidateSpy).toHaveBeenNthCalledWith(1, {
      queryKey: websiteKeys.all,
    })
    expect(invalidateSpy).toHaveBeenNthCalledWith(2, {
      queryKey: ["targets"],
    })
    expect(invalidateSpy).toHaveBeenNthCalledWith(3, {
      queryKey: ["scans"],
    })
  })

  it("批量删除网站失败时保留错误码映射与 fallback key", async () => {
    websiteServiceMocks.bulkDeleteWebSites.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useBulkDeleteWebSites())

    await act(async () => {
      await expect(result.current.mutateAsync([11])).rejects.toBeDefined()
    })

    expect(websiteServiceMocks.bulkDeleteWebSites).toHaveBeenCalled()
    expect(websiteServiceMocks.bulkDeleteWebSites.mock.calls[0]?.[0]).toEqual([11])
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.batchDeleting",
      {},
      "bulk-delete-websites"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("bulk-delete-websites")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.asset.website.delete.error"
    )
  })

  it("删除网站成功时保留 success 提示与失效范围", async () => {
    websiteServiceMocks.deleteWebSite.mockResolvedValue(undefined)

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useDeleteWebSite(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(11)
    })

    expect(websiteServiceMocks.deleteWebSite).toHaveBeenCalled()
    expect(websiteServiceMocks.deleteWebSite.mock.calls[0]?.[0]).toBe(11)
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.deleting",
      {},
      "delete-website-11"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("delete-website-11")
    expect(toastMocks.success).toHaveBeenCalledWith("toast.asset.website.delete.success")
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: websiteKeys.all })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["targets"] })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["scans"] })
  })

  it("删除网站失败时保留 loading 生命周期与 fallback 错误映射", async () => {
    websiteServiceMocks.deleteWebSite.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useDeleteWebSite())

    await act(async () => {
      await expect(result.current.mutateAsync(11)).rejects.toBeDefined()
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.deleting",
      {},
      "delete-website-11"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("delete-website-11")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.asset.website.delete.error"
    )
  })
})
