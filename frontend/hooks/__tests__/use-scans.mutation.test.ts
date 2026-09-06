import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  scanKeys,
  useBatchStopScans,
  useBulkDeleteScans,
  useDeleteScan,
  useInitiateScan,
  useQuickScan,
  useStopScan,
} from "@/hooks/use-scans"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const scanServiceMocks = vi.hoisted(() => ({
  getScans: vi.fn(),
  getScan: vi.fn(),
  getScanStatistics: vi.fn(),
  quickScan: vi.fn(),
  initiateScan: vi.fn(),
  deleteScan: vi.fn(),
  bulkDeleteScans: vi.fn(),
  batchStopScans: vi.fn(),
  stopScan: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/scan.service", () => ({
  getScans: scanServiceMocks.getScans,
  getScan: scanServiceMocks.getScan,
  getScanStatistics: scanServiceMocks.getScanStatistics,
  quickScan: scanServiceMocks.quickScan,
  initiateScan: scanServiceMocks.initiateScan,
  deleteScan: scanServiceMocks.deleteScan,
  bulkDeleteScans: scanServiceMocks.bulkDeleteScans,
  batchStopScans: scanServiceMocks.batchStopScans,
  stopScan: scanServiceMocks.stopScan,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

describe("use-scans mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("快速扫描成功时保持成功提示与查询失效范围", async () => {
    scanServiceMocks.quickScan.mockResolvedValue({
      scans: [{ id: 1 }, { id: 2 }],
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")

    const { result } = renderHookWithProviders(() => useQuickScan(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        targets: [{ name: "example.com" }],
        configuration: "name: quick",
        scanWorkflow: "subdomain_discovery",
        inputSource: "scanSnapshot",
      })
    })

    expect(scanServiceMocks.quickScan).toHaveBeenCalled()
    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.scan.quick.success",
      { count: 2 },
      "quick-scan"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scanKeys.all })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scanKeys.statistics() })
  })

  it("快速扫描失败时保持错误码映射和回退 key", async () => {
    scanServiceMocks.quickScan.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "RATE_LIMITED",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useQuickScan())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          targets: [{ name: "blocked.com" }],
          configuration: "name: quick",
          scanWorkflow: "subdomain_discovery",
          inputSource: "scanSnapshot",
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "RATE_LIMITED",
      "toast.scan.quick.error",
      "quick-scan"
    )
  })

  it("发起扫描成功时保持成功提示与失效范围", async () => {
    scanServiceMocks.initiateScan.mockResolvedValue({
      id: 11,
      status: "queued",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useInitiateScan(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        targetId: 1,
        configuration: "name: nightly",
        scanWorkflow: "subdomain_discovery",
        inputSource: "scanSnapshot",
      })
    })

    expect(scanServiceMocks.initiateScan).toHaveBeenCalled()
    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.scan.initiate.success",
      undefined,
      "initiate-scan-1"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scanKeys.all })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scanKeys.statistics() })
  })

  it("发起扫描失败时保持错误码映射和回退 key", async () => {
    scanServiceMocks.initiateScan.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useInitiateScan())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          targetId: 2,
          configuration: "name: nightly",
          scanWorkflow: "subdomain_discovery",
          inputSource: "scanSnapshot",
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.scan.initiate.error",
      "initiate-scan-2"
    )
  })

  it("删除扫描成功时保持成功提示与失效范围", async () => {
    scanServiceMocks.deleteScan.mockResolvedValue({
      message: "ok",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useDeleteScan(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(9)
    })

    expect(scanServiceMocks.deleteScan).toHaveBeenCalledWith(9)
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.deleting",
      {},
      "delete-scan-9"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalledWith("delete-scan-9")
    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.scan.delete.success",
      { name: "Scan #9" },
      "delete-scan-9"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scanKeys.all })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scanKeys.statistics() })
  })

  it("删除扫描失败时保持错误码映射和回退 key", async () => {
    scanServiceMocks.deleteScan.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useDeleteScan())

    await act(async () => {
      await expect(result.current.mutateAsync(9)).rejects.toBeDefined()
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.deleting",
      {},
      "delete-scan-9"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalledWith("delete-scan-9")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.deleteFailed",
      "delete-scan-9"
    )
  })

  it("批量删除扫描成功时保持成功提示与失效范围", async () => {
    scanServiceMocks.bulkDeleteScans.mockResolvedValue({
      deletedCount: 2,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useBulkDeleteScans(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync([1, 2])
    })

    expect(scanServiceMocks.bulkDeleteScans).toHaveBeenCalledWith([1, 2])
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.batchDeleting",
      {},
      "bulk-delete-scans"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalledWith("bulk-delete-scans")
    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.scan.delete.bulkSuccess",
      { count: 2 },
      "bulk-delete-scans"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scanKeys.all })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scanKeys.statistics() })
  })

  it("批量删除扫描失败时保持错误码映射和回退 key", async () => {
    scanServiceMocks.bulkDeleteScans.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useBulkDeleteScans())

    await act(async () => {
      await expect(result.current.mutateAsync([1, 2])).rejects.toBeDefined()
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.batchDeleting",
      {},
      "bulk-delete-scans"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalledWith("bulk-delete-scans")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.bulkDeleteFailed",
      "bulk-delete-scans"
    )
  })

  it("停止扫描成功时保持成功提示与失效范围", async () => {
    scanServiceMocks.stopScan.mockResolvedValue({
      revokedTaskCount: 3,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useStopScan(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(5)
    })

    expect(scanServiceMocks.stopScan).toHaveBeenCalledWith(5)
    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.scan.stop.success",
      { count: 3 },
      "stop-scan-5"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scanKeys.all })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scanKeys.statistics() })
  })

  it("停止扫描失败时保持错误码映射和回退 key", async () => {
    scanServiceMocks.stopScan.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useStopScan())

    await act(async () => {
      await expect(result.current.mutateAsync(5)).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.stopFailed",
      "stop-scan-5"
    )
  })

  it("批量停止混合选择时提示提交和跳过计数并失效历史查询", async () => {
    scanServiceMocks.batchStopScans.mockResolvedValue({
      stoppedCount: 2,
      skippedCount: 1,
      revokedTaskCount: 2,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useBatchStopScans(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync([1, 2, 3])
    })

    expect(scanServiceMocks.batchStopScans).toHaveBeenCalledWith([1, 2, 3])
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.updating",
      {},
      "batch-stop-scans"
    )
    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.scan.stop.batchSuccess",
      { stoppedCount: 2, skippedCount: 1 },
      "batch-stop-scans"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scanKeys.all })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scanKeys.statistics() })
  })

  it("批量停止失败时保留稳定错误反馈", async () => {
    scanServiceMocks.batchStopScans.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useBatchStopScans())

    await act(async () => {
      await expect(result.current.mutateAsync([1, 2])).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.scan.stop.batchError",
      "batch-stop-scans"
    )
  })
})
