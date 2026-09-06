import { act, renderHook } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { useScanHistoryActions } from "@/components/scan/history/scan-history-list-state"
import type { ScanRecord } from "@/types/scan.types"

const mutationMocks = vi.hoisted(() => ({
  delete: { mutateAsync: vi.fn() },
  bulkDelete: { mutateAsync: vi.fn() },
  stop: { mutateAsync: vi.fn() },
  batchStop: { mutateAsync: vi.fn(), isPending: false },
  loadDetail: vi.fn(),
}))

vi.mock("@/hooks/use-scans", () => ({
  useDeleteScan: () => mutationMocks.delete,
  useBulkDeleteScans: () => mutationMocks.bulkDelete,
  useStopScan: () => mutationMocks.stop,
  useBatchStopScans: () => mutationMocks.batchStop,
  useLoadScanDetail: () => mutationMocks.loadDetail,
}))

function scan(id: number, status: ScanRecord["status"]): ScanRecord {
  return {
    id,
    targetId: id,
    plannedEngineIds: [],
    triggerType: "manual",
    inputSource: "scanSnapshot",
    createdAt: "2026-08-18T00:00:00Z",
    status,
    progress: status === "succeeded" ? 100 : 20,
  }
}

describe("useScanHistoryActions batch stop", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mutationMocks.batchStop.isPending = false
    mutationMocks.batchStop.mutateAsync.mockResolvedValue({
      stoppedCount: 1,
      skippedCount: 1,
      revokedTaskCount: 1,
    })
  })

  it("keeps Stop unavailable for a terminal-only selection", async () => {
    const clearSelection = vi.fn()
    const { result } = renderHook(() => useScanHistoryActions({
      selectedScans: [scan(1, "succeeded"), scan(2, "failed")],
      onSelectionClear: clearSelection,
    }))

    expect(result.current.activeSelectedScans).toHaveLength(0)
    expect(result.current.terminalSelectedCount).toBe(2)
    expect(result.current.canBatchStop).toBe(false)

    act(() => result.current.handleBatchStop())
    expect(result.current.batchStopDialogOpen).toBe(false)

    await act(async () => {
      await result.current.confirmBatchStop()
    })
    expect(mutationMocks.batchStop.mutateAsync).not.toHaveBeenCalled()
    expect(clearSelection).not.toHaveBeenCalled()
  })

  it("confirms mixed selections with all IDs and clears only after success", async () => {
    const clearSelection = vi.fn()
    const { result } = renderHook(() => useScanHistoryActions({
      selectedScans: [scan(1, "running"), scan(2, "cancelled")],
      onSelectionClear: clearSelection,
    }))

    act(() => result.current.handleBatchStop())
    expect(result.current.batchStopDialogOpen).toBe(true)

    await act(async () => {
      await result.current.confirmBatchStop()
    })

    expect(mutationMocks.batchStop.mutateAsync).toHaveBeenCalledWith([1, 2])
    expect(clearSelection).toHaveBeenCalledTimes(1)
    expect(result.current.batchStopDialogOpen).toBe(false)
  })

  it("retains selection when the batch request fails", async () => {
    mutationMocks.batchStop.mutateAsync.mockRejectedValueOnce(new Error("network"))
    const clearSelection = vi.fn()
    const { result } = renderHook(() => useScanHistoryActions({
      selectedScans: [scan(1, "pending")],
      onSelectionClear: clearSelection,
    }))

    act(() => result.current.handleBatchStop())
    await act(async () => {
      await result.current.confirmBatchStop()
    })

    expect(clearSelection).not.toHaveBeenCalled()
    expect(mutationMocks.batchStop.mutateAsync).toHaveBeenCalledWith([1])
  })
})
