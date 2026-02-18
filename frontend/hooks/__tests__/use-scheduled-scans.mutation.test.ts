import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  scheduledScanKeys,
  useCreateScheduledScan,
  useDeleteScheduledScan,
  useToggleScheduledScan,
  useUpdateScheduledScan,
} from "@/hooks/use-scheduled-scans"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const scheduledScanServiceMocks = vi.hoisted(() => ({
  getScheduledScans: vi.fn(),
  getScheduledScan: vi.fn(),
  createScheduledScan: vi.fn(),
  updateScheduledScan: vi.fn(),
  deleteScheduledScan: vi.fn(),
  toggleScheduledScan: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/scheduled-scan.service", () => ({
  getScheduledScans: scheduledScanServiceMocks.getScheduledScans,
  getScheduledScan: scheduledScanServiceMocks.getScheduledScan,
  createScheduledScan: scheduledScanServiceMocks.createScheduledScan,
  updateScheduledScan: scheduledScanServiceMocks.updateScheduledScan,
  deleteScheduledScan: scheduledScanServiceMocks.deleteScheduledScan,
  toggleScheduledScan: scheduledScanServiceMocks.toggleScheduledScan,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

describe("use-scheduled-scans mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("创建定时扫描成功时保持成功提示和失效范围", async () => {
    scheduledScanServiceMocks.createScheduledScan.mockResolvedValue({
      message: "ok",
      scheduledScan: {
        id: 1,
        name: "daily-scan",
      },
    })
    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useCreateScheduledScan(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        name: "daily-scan",
        configuration: "name: daily",
        engineIds: [1],
        engineNames: ["nuclei"],
        targetId: 9,
        cronExpression: "0 9 * * *",
        isEnabled: true,
      })
    })

    expect(toastMocks.success).toHaveBeenCalledWith("toast.scheduledScan.create.success")
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: scheduledScanKeys.all,
    })
  })

  it("创建定时扫描失败时保持错误码映射与回退 key", async () => {
    scheduledScanServiceMocks.createScheduledScan.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "CONFLICT",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useCreateScheduledScan())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          name: "daily-scan",
          configuration: "name: daily",
          engineIds: [1],
          engineNames: ["nuclei"],
          targetId: 9,
          cronExpression: "0 9 * * *",
          isEnabled: true,
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "CONFLICT",
      "toast.scheduledScan.create.error"
    )
  })

  it("更新定时扫描成功时保持成功提示和失效范围", async () => {
    scheduledScanServiceMocks.updateScheduledScan.mockResolvedValue({
      message: "ok",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUpdateScheduledScan(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        id: 5,
        data: {
          name: "weekly-scan",
        },
      })
    })

    expect(scheduledScanServiceMocks.updateScheduledScan).toHaveBeenCalledWith(5, {
      name: "weekly-scan",
    })
    expect(toastMocks.success).toHaveBeenCalledWith("toast.scheduledScan.update.success")
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scheduledScanKeys.all })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scheduledScanKeys.details() })
  })

  it("更新定时扫描失败时保持错误码映射与回退 key", async () => {
    scheduledScanServiceMocks.updateScheduledScan.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useUpdateScheduledScan())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          id: 5,
          data: {
            name: "weekly-scan",
          },
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.scheduledScan.update.error"
    )
  })

  it("删除定时扫描成功时保持成功提示和失效范围", async () => {
    scheduledScanServiceMocks.deleteScheduledScan.mockResolvedValue({
      message: "ok",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useDeleteScheduledScan(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(12)
    })

    expect(scheduledScanServiceMocks.deleteScheduledScan).toHaveBeenCalledWith(12)
    expect(toastMocks.success).toHaveBeenCalledWith("toast.scheduledScan.delete.success")
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scheduledScanKeys.all })
  })

  it("删除定时扫描失败时保持错误码映射与回退 key", async () => {
    scheduledScanServiceMocks.deleteScheduledScan.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useDeleteScheduledScan())

    await act(async () => {
      await expect(result.current.mutateAsync(12)).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.scheduledScan.delete.error"
    )
  })

  it("切换状态成功时根据 isEnabled 提示不同文案", async () => {
    scheduledScanServiceMocks.toggleScheduledScan.mockResolvedValue({
      message: "ok",
    })

    const { result, rerender } = renderHookWithProviders(() => useToggleScheduledScan())

    await act(async () => {
      await result.current.mutateAsync({
        id: 3,
        isEnabled: true,
      })
    })

    expect(toastMocks.success).toHaveBeenCalledWith("toast.scheduledScan.toggle.enabled")

    rerender()

    await act(async () => {
      await result.current.mutateAsync({
        id: 3,
        isEnabled: false,
      })
    })

    expect(toastMocks.success).toHaveBeenCalledWith("toast.scheduledScan.toggle.disabled")
  })

  it("切换状态失败时使用错误码提示", async () => {
    scheduledScanServiceMocks.toggleScheduledScan.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useToggleScheduledScan())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          id: 3,
          isEnabled: false,
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith("FORBIDDEN")
  })

  it("切换状态失败且无错误码时走 fallback 提示", async () => {
    scheduledScanServiceMocks.toggleScheduledScan.mockRejectedValue({
      response: {
        data: {
          error: {},
        },
      },
    })

    const { result } = renderHookWithProviders(() => useToggleScheduledScan())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          id: 6,
          isEnabled: false,
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.error).toHaveBeenCalledWith("toast.scheduledScan.toggle.error")
  })
})
