import { act, renderHook } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import {
  SCAN_RUNTIME_DETAIL_REFRESH_MS,
  useScanOverviewState,
} from "@/components/scan/history/scan-overview-state"

const stateMocks = vi.hoisted(() => ({
  scan: {
    status: "running",
    cachedStats: {},
    runtimeTasks: [],
  } as Record<string, unknown>,
  isFetching: false,
  refetch: vi.fn(),
}))

vi.mock("@/hooks/use-scans", () => ({
  useScan: () => ({
    data: stateMocks.scan,
    isLoading: false,
    isFetching: stateMocks.isFetching,
    error: null,
    refetch: stateMocks.refetch,
  }),
}))

vi.mock("@/hooks/use-task-progress-logs", () => ({
  useTaskProgressLogs: () => ({ logs: [], loading: false }),
}))

describe("useScanOverviewState runtime detail refresh", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.clearAllMocks()
    stateMocks.scan = { status: "running", cachedStats: {}, runtimeTasks: [] }
    stateMocks.isFetching = false
    stateMocks.refetch.mockResolvedValue(undefined)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it("每三秒刷新打开的运行中详情", async () => {
    renderHook(() => useScanOverviewState({
      scanId: 7,
      t: (key) => key,
      refreshEnabled: true,
    }))

    await act(async () => {
      await vi.advanceTimersByTimeAsync(SCAN_RUNTIME_DETAIL_REFRESH_MS)
    })

    expect(stateMocks.refetch).toHaveBeenCalledTimes(1)
  })

  it("请求进行中时不启动重叠的详情刷新", async () => {
    stateMocks.isFetching = true

    renderHook(() => useScanOverviewState({
      scanId: 7,
      t: (key) => key,
      refreshEnabled: true,
    }))

    await act(async () => {
      await vi.advanceTimersByTimeAsync(SCAN_RUNTIME_DETAIL_REFRESH_MS * 2)
    })

    expect(stateMocks.refetch).not.toHaveBeenCalled()
  })

  it("关闭抽屉或进入终态后停止详情刷新", async () => {
    const { rerender, unmount } = renderHook(
      ({ refreshEnabled }) => useScanOverviewState({
        scanId: 7,
        t: (key) => key,
        refreshEnabled,
      }),
      { initialProps: { refreshEnabled: true } }
    )

    rerender({ refreshEnabled: false })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(SCAN_RUNTIME_DETAIL_REFRESH_MS)
    })
    expect(stateMocks.refetch).not.toHaveBeenCalled()

    stateMocks.scan = { status: "succeeded", cachedStats: {}, runtimeTasks: [] }
    rerender({ refreshEnabled: true })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(SCAN_RUNTIME_DETAIL_REFRESH_MS)
    })
    expect(stateMocks.refetch).not.toHaveBeenCalled()

    unmount()
  })
})
