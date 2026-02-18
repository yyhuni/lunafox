import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  useBulkDeleteDirectories,
  useDeleteDirectory,
  directoryKeys,
  useBulkCreateDirectories,
} from "@/hooks/use-directories"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const directoryServiceMocks = vi.hoisted(() => ({
  getTargetDirectories: vi.fn(),
  getScanDirectories: vi.fn(),
  deleteDirectory: vi.fn(),
  bulkDeleteDirectories: vi.fn(),
  bulkDelete: vi.fn(),
  bulkCreateDirectories: vi.fn(),
  exportDirectoriesByTargetId: vi.fn(),
  exportDirectoriesByScanId: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/directory.service", () => ({
  DirectoryService: directoryServiceMocks,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

describe("use-directories mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("批量创建目录成功时保留成功提示与目录列表失效范围", async () => {
    directoryServiceMocks.bulkCreateDirectories.mockResolvedValue({
      message: "ok",
      createdCount: 3,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useBulkCreateDirectories(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        targetId: 8,
        urls: ["https://example.com/admin", "https://example.com/login"],
      })
    })

    expect(directoryServiceMocks.bulkCreateDirectories).toHaveBeenCalledWith(8, [
      "https://example.com/admin",
      "https://example.com/login",
    ])
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.batchCreating",
      {},
      "bulk-create-directories"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("bulk-create-directories")
    expect(toastMocks.success).toHaveBeenCalledWith("toast.asset.directory.create.success", {
      count: 3,
    })
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: directoryKeys.all,
      exact: false,
      refetchType: "active",
    })
  })

  it("批量创建目录 createdCount 为 0 时走 partialSuccess 警告路径", async () => {
    directoryServiceMocks.bulkCreateDirectories.mockResolvedValue({
      message: "ok",
      createdCount: 0,
    })

    const { result } = renderHookWithProviders(() => useBulkCreateDirectories())

    await act(async () => {
      await result.current.mutateAsync({
        targetId: 8,
        urls: ["https://example.com/admin"],
      })
    })

    expect(toastMocks.warning).toHaveBeenCalledWith(
      "toast.asset.directory.create.partialSuccess",
      { success: 0, skipped: 0 }
    )
    expect(toastMocks.success).not.toHaveBeenCalled()
  })

  it("批量删除目录成功时保留 bulkSuccess 提示与失效范围", async () => {
    directoryServiceMocks.bulkDeleteDirectories.mockResolvedValue({
      message: "ok",
      deletedCount: 2,
      requestedIds: [31, 32],
      cascadeDeleted: {},
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useBulkDeleteDirectories(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync([31, 32])
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.batchDeleting",
      {},
      "bulk-delete-directories"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("bulk-delete-directories")
    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.asset.directory.delete.bulkSuccess",
      { count: 2 }
    )
    expect(invalidateSpy).toHaveBeenNthCalledWith(1, {
      queryKey: directoryKeys.all,
    })
    expect(invalidateSpy).toHaveBeenNthCalledWith(2, {
      queryKey: ["targets"],
    })
    expect(invalidateSpy).toHaveBeenNthCalledWith(3, {
      queryKey: ["scans"],
    })
  })

  it("批量删除目录失败时保留错误码映射与 fallback key", async () => {
    directoryServiceMocks.bulkDeleteDirectories.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useBulkDeleteDirectories())

    await act(async () => {
      await expect(result.current.mutateAsync([31])).rejects.toBeDefined()
    })

    expect(directoryServiceMocks.bulkDeleteDirectories).toHaveBeenCalled()
    expect(directoryServiceMocks.bulkDeleteDirectories.mock.calls[0]?.[0]).toEqual([31])
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.batchDeleting",
      {},
      "bulk-delete-directories"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("bulk-delete-directories")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.asset.directory.delete.error"
    )
  })

  it("删除目录成功时保留 success 提示与失效范围", async () => {
    directoryServiceMocks.deleteDirectory.mockResolvedValue(undefined)

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useDeleteDirectory(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(31)
    })

    expect(directoryServiceMocks.deleteDirectory).toHaveBeenCalled()
    expect(directoryServiceMocks.deleteDirectory.mock.calls[0]?.[0]).toBe(31)
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.deleting",
      {},
      "delete-directory-31"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("delete-directory-31")
    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.asset.directory.delete.success"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: directoryKeys.all })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["targets"] })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["scans"] })
  })

  it("删除目录失败时保留错误码映射与 fallback key", async () => {
    directoryServiceMocks.deleteDirectory.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useDeleteDirectory())

    await act(async () => {
      await expect(result.current.mutateAsync(31)).rejects.toBeDefined()
    })

    expect(directoryServiceMocks.deleteDirectory).toHaveBeenCalled()
    expect(directoryServiceMocks.deleteDirectory.mock.calls[0]?.[0]).toBe(31)
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.deleting",
      {},
      "delete-directory-31"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("delete-directory-31")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.asset.directory.delete.error"
    )
  })
})
