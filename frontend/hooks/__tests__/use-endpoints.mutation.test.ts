import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  endpointKeys,
  useBatchDeleteEndpoints,
  useBulkCreateEndpoints,
  useCreateEndpoint,
  useDeleteEndpoint,
} from "@/hooks/use-endpoints"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const endpointServiceMocks = vi.hoisted(() => ({
  bulkDelete: vi.fn(),
  bulkCreateEndpoints: vi.fn(),
  getEndpointById: vi.fn(),
  getEndpoints: vi.fn(),
  getEndpointsByTargetId: vi.fn(),
  getEndpointsByScanId: vi.fn(),
  createEndpoints: vi.fn(),
  deleteEndpoint: vi.fn(),
  batchDeleteEndpoints: vi.fn(),
  exportEndpointsByTargetId: vi.fn(),
  exportEndpointsByScanId: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/endpoint.service", () => ({
  EndpointService: endpointServiceMocks,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

describe("use-endpoints mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("创建 endpoint 出现已存在项时走 partialSuccess 且失效 endpoints", async () => {
    endpointServiceMocks.createEndpoints.mockResolvedValue({
      message: "ok",
      createdCount: 1,
      existedCount: 2,
      skippedCount: 0,
      totalReceived: 3,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useCreateEndpoint(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        endpoints: [
          { url: "https://a.example.com", method: "GET" },
          { url: "https://b.example.com", method: "GET" },
        ],
      })
    })

    expect(endpointServiceMocks.createEndpoints).toHaveBeenCalledWith({
      endpoints: [
        { url: "https://a.example.com", method: "GET" },
        { url: "https://b.example.com", method: "GET" },
      ],
    })
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.creating",
      {},
      "create-endpoint"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("create-endpoint")
    expect(toastMocks.warning).toHaveBeenCalledWith(
      "toast.asset.endpoint.create.partialSuccess",
      { success: 1, skipped: 2 }
    )
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: endpointKeys.all })
  })

  it("创建 endpoint 成功时保留 success 提示与失效范围", async () => {
    endpointServiceMocks.createEndpoints.mockResolvedValue({
      message: "ok",
      createdCount: 2,
      existedCount: 0,
      skippedCount: 0,
      totalReceived: 2,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useCreateEndpoint(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        endpoints: [
          { url: "https://c.example.com", method: "GET" },
          { url: "https://d.example.com", method: "GET" },
        ],
      })
    })

    expect(endpointServiceMocks.createEndpoints).toHaveBeenCalledWith({
      endpoints: [
        { url: "https://c.example.com", method: "GET" },
        { url: "https://d.example.com", method: "GET" },
      ],
    })
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.creating",
      {},
      "create-endpoint"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("create-endpoint")
    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.asset.endpoint.create.success",
      { count: 2 }
    )
    expect(toastMocks.warning).not.toHaveBeenCalled()
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: endpointKeys.all })
  })

  it("批量创建 endpoint 成功时按 target 与 endpoints 维度失效", async () => {
    endpointServiceMocks.bulkCreateEndpoints.mockResolvedValue({
      message: "ok",
      createdCount: 2,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useBulkCreateEndpoints(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        targetId: 9,
        urls: ["https://api.example.com/health", "https://api.example.com/login"],
      })
    })

    expect(endpointServiceMocks.bulkCreateEndpoints).toHaveBeenCalledWith(9, [
      "https://api.example.com/health",
      "https://api.example.com/login",
    ])
    expect(toastMocks.success).toHaveBeenCalledWith("toast.asset.endpoint.create.success", {
      count: 2,
    })
    expect(invalidateSpy).toHaveBeenNthCalledWith(1, {
      queryKey: endpointKeys.byTarget(9, {}),
      exact: false,
      refetchType: "active",
    })
    expect(invalidateSpy).toHaveBeenNthCalledWith(2, {
      queryKey: endpointKeys.all,
      exact: false,
      refetchType: "active",
    })
  })

  it("批量创建 endpoint createdCount 为 0 时走 partialSuccess 警告路径", async () => {
    endpointServiceMocks.bulkCreateEndpoints.mockResolvedValue({
      message: "ok",
      createdCount: 0,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useBulkCreateEndpoints(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        targetId: 9,
        urls: ["https://api.example.com/health"],
      })
    })

    expect(endpointServiceMocks.bulkCreateEndpoints).toHaveBeenCalledWith(9, [
      "https://api.example.com/health",
    ])
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.batchCreating",
      {},
      "bulk-create-endpoints"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("bulk-create-endpoints")
    expect(toastMocks.warning).toHaveBeenCalledWith(
      "toast.asset.endpoint.create.partialSuccess",
      { success: 0, skipped: 0 }
    )
    expect(toastMocks.success).not.toHaveBeenCalled()
    expect(invalidateSpy).toHaveBeenNthCalledWith(1, {
      queryKey: endpointKeys.byTarget(9, {}),
      exact: false,
      refetchType: "active",
    })
    expect(invalidateSpy).toHaveBeenNthCalledWith(2, {
      queryKey: endpointKeys.all,
      exact: false,
      refetchType: "active",
    })
  })

  it("删除 endpoint 失败时保留错误码映射与 fallback key", async () => {
    endpointServiceMocks.deleteEndpoint.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useDeleteEndpoint())

    await act(async () => {
      await expect(result.current.mutateAsync(55)).rejects.toBeDefined()
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.deleting",
      {},
      "delete-endpoint-55"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("delete-endpoint-55")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.asset.endpoint.delete.error"
    )
  })

  it("创建 endpoint 失败时保留错误码映射与 fallback key", async () => {
    endpointServiceMocks.createEndpoints.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useCreateEndpoint())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          endpoints: [{ url: "https://e.example.com", method: "GET" }],
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.creating",
      {},
      "create-endpoint"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("create-endpoint")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.asset.endpoint.create.error"
    )
  })

  it("删除 endpoint 成功时保留 success 提示与失效范围", async () => {
    endpointServiceMocks.deleteEndpoint.mockResolvedValue(undefined)

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useDeleteEndpoint(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(55)
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.deleting",
      {},
      "delete-endpoint-55"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("delete-endpoint-55")
    expect(toastMocks.success).toHaveBeenCalledWith("toast.asset.endpoint.delete.success")
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: endpointKeys.all })
  })

  it("批量删除 endpoint 成功时保留 bulkSuccess 提示与失效范围", async () => {
    endpointServiceMocks.batchDeleteEndpoints.mockResolvedValue({
      message: "ok",
      deletedCount: 2,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useBatchDeleteEndpoints(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({ endpointIds: [1, 2] })
    })

    expect(endpointServiceMocks.batchDeleteEndpoints).toHaveBeenCalled()
    expect(endpointServiceMocks.batchDeleteEndpoints.mock.calls[0][0]).toEqual({
      endpointIds: [1, 2],
    })
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.batchDeleting",
      {},
      "batch-delete-endpoints"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("batch-delete-endpoints")
    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.asset.endpoint.delete.bulkSuccess",
      { count: 2 }
    )
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: endpointKeys.all })
  })

  it("批量删除 endpoint 失败时保留错误码映射与 fallback key", async () => {
    endpointServiceMocks.batchDeleteEndpoints.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useBatchDeleteEndpoints())

    await act(async () => {
      await expect(
        result.current.mutateAsync({ endpointIds: [1] })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.batchDeleting",
      {},
      "batch-delete-endpoints"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("batch-delete-endpoints")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.asset.endpoint.delete.error"
    )
  })
})
