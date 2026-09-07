import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  scheduledScanKeys,
  scheduledScanOverviewKey,
  useBatchDeleteScheduledScans,
  useBatchUpdateScheduledScanStatus,
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
	batchDeleteScheduledScans: vi.fn(),
	batchUpdateScheduledScanStatus: vi.fn(),
	toggleScheduledScan: vi.fn(),
	getScheduledScanOverviewSummary: vi.fn(),
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
	batchDeleteScheduledScans: scheduledScanServiceMocks.batchDeleteScheduledScans,
	batchUpdateScheduledScanStatus: scheduledScanServiceMocks.batchUpdateScheduledScanStatus,
	toggleScheduledScan: scheduledScanServiceMocks.toggleScheduledScan,
	getScheduledScanOverviewSummary: scheduledScanServiceMocks.getScheduledScanOverviewSummary,
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
      id: 1,
      name: "scheduledScans/1",
      displayName: "daily-scan",
    })
    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useCreateScheduledScan(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        displayName: "daily-scan",
        configuration: "name: daily",
        scanWorkflow: "subdomain_discovery",
        targetId: 9,
        cronExpression: "0 9 * * *",
        isEnabled: true,
        inputSource: "scanSnapshot",
      })
    })

    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.scheduledScan.create.success",
      undefined,
      "create-scheduled-scan"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: scheduledScanKeys.all,
    })
		expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scheduledScanOverviewKey })
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
          displayName: "daily-scan",
          configuration: "name: daily",
          scanWorkflow: "subdomain_discovery",
          targetId: 9,
          cronExpression: "0 9 * * *",
          isEnabled: true,
          inputSource: "scanSnapshot",
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "CONFLICT",
      "toast.scheduledScan.create.error",
      "create-scheduled-scan"
    )
  })

  it("更新定时扫描成功时保持成功提示和失效范围", async () => {
    scheduledScanServiceMocks.updateScheduledScan.mockResolvedValue({
      id: 5,
      name: "scheduledScans/5",
      displayName: "weekly-scan",
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
          displayName: "weekly-scan",
        },
      })
    })

    expect(scheduledScanServiceMocks.updateScheduledScan).toHaveBeenCalledWith(5, {
      displayName: "weekly-scan",
    })
    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.scheduledScan.update.success",
      undefined,
      "update-scheduled-scan-5"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scheduledScanKeys.all })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scheduledScanKeys.details() })
		expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scheduledScanOverviewKey })
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
            displayName: "weekly-scan",
          },
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.scheduledScan.update.error",
      "update-scheduled-scan-5"
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
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.deleting",
      {},
      "delete-scheduled-scan-12"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalledWith("delete-scheduled-scan-12")
    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.scheduledScan.delete.success",
      undefined,
      "delete-scheduled-scan-12"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scheduledScanKeys.all })
		expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scheduledScanOverviewKey })
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

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.deleting",
      {},
      "delete-scheduled-scan-12"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalledWith("delete-scheduled-scan-12")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.scheduledScan.delete.error",
      "delete-scheduled-scan-12"
    )
  })

	it("批量删除定时扫描成功时保持单个 loading 生命周期、成功提示和失效范围", async () => {
    scheduledScanServiceMocks.batchDeleteScheduledScans.mockResolvedValue({
      message: "ok",
      deletedCount: 2,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useBatchDeleteScheduledScans(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync([12, 13])
    })

    expect(scheduledScanServiceMocks.batchDeleteScheduledScans).toHaveBeenCalledWith([12, 13])
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.batchDeleting",
      {},
      "batch-delete-scheduled-scans"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalledWith("batch-delete-scheduled-scans")
    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.scheduledScan.delete.bulkSuccess",
      { count: 2 },
      "batch-delete-scheduled-scans"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scheduledScanKeys.all })
		expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scheduledScanOverviewKey })
	})

	it("批量状态更新成功时发送显式目标状态并失效列表与概览", async () => {
		scheduledScanServiceMocks.batchUpdateScheduledScanStatus.mockResolvedValue({
			updatedCount: 2,
		})

		const queryClient = createTestQueryClient()
		const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
		const { result } = renderHookWithProviders(() => useBatchUpdateScheduledScanStatus(), {
			queryClient,
		})

		await act(async () => {
			await result.current.mutateAsync({ ids: [12, 13], isEnabled: false })
		})

		expect(scheduledScanServiceMocks.batchUpdateScheduledScanStatus).toHaveBeenCalledWith({
			ids: [12, 13],
			isEnabled: false,
		})
		expect(toastMocks.loading).toHaveBeenCalledWith(
			"common.status.updating",
			{},
			"batch-update-scheduled-scan-status"
		)
		expect(toastMocks.dismiss).not.toHaveBeenCalledWith("batch-update-scheduled-scan-status")
		expect(toastMocks.success).toHaveBeenCalledWith(
			"toast.scheduledScan.batchStatus.disabled",
			{ count: 2 },
			"batch-update-scheduled-scan-status"
		)
		expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scheduledScanKeys.all })
		expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scheduledScanOverviewKey })
	})

	it("批量状态更新失败时保留错误码回退和 loading 生命周期", async () => {
		scheduledScanServiceMocks.batchUpdateScheduledScanStatus.mockRejectedValue({
			response: { data: { error: { code: "FORBIDDEN" } } },
		})

		const { result } = renderHookWithProviders(() => useBatchUpdateScheduledScanStatus())

		await act(async () => {
			await expect(
				result.current.mutateAsync({ ids: [12], isEnabled: true })
			).rejects.toBeDefined()
		})

		expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
			"FORBIDDEN",
			"toast.scheduledScan.batchStatus.error",
			"batch-update-scheduled-scan-status"
		)
		expect(toastMocks.dismiss).not.toHaveBeenCalledWith("batch-update-scheduled-scan-status")
	})

	it("切换状态成功时根据 isEnabled 提示不同文案", async () => {
    scheduledScanServiceMocks.toggleScheduledScan.mockResolvedValue({
      message: "ok",
    })

		const queryClient = createTestQueryClient()
		const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
		const { result, rerender } = renderHookWithProviders(() => useToggleScheduledScan(), { queryClient })

    await act(async () => {
      await result.current.mutateAsync({
        id: 3,
        isEnabled: true,
      })
    })

    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.scheduledScan.toggle.enabled",
      undefined,
      "toggle-scheduled-scan-3"
    )
		expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: scheduledScanOverviewKey })

    rerender()

    await act(async () => {
      await result.current.mutateAsync({
        id: 3,
        isEnabled: false,
      })
    })

    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.scheduledScan.toggle.disabled",
      undefined,
      "toggle-scheduled-scan-3"
    )
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

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      undefined,
      "toggle-scheduled-scan-3"
    )
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

    expect(toastMocks.error).toHaveBeenCalledWith(
      "toast.scheduledScan.toggle.error",
      undefined,
      "toggle-scheduled-scan-6"
    )
  })
})
