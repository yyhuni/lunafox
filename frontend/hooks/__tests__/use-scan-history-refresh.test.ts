import { act } from "@testing-library/react"
import type { QueryClient } from "@tanstack/react-query"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { scanKeys } from "@/hooks/use-scans"
import {
  SCAN_HISTORY_AUTO_REFRESH_MS,
  useScanHistoryRefresh,
} from "@/hooks/use-scan-history-refresh"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"

const scanHooksMocks = vi.hoisted(() => ({
  useScanStatistics: vi.fn(),
}))

vi.mock("@/hooks/use-scans", async () => {
  const actual = await vi.importActual<typeof import("@/hooks/use-scans")>("@/hooks/use-scans")
  return {
    ...actual,
    useScanStatistics: scanHooksMocks.useScanStatistics,
  }
})

function deferred() {
  let resolve!: () => void
  const promise = new Promise<void>((resolvePromise) => {
    resolve = resolvePromise
  })

  return { promise, resolve }
}

function mockRefetches(queryClient: QueryClient) {
  const requests = [deferred(), deferred()]
  let requestIndex = 0
  const refetch = vi.spyOn(queryClient, "refetchQueries").mockImplementation(() => {
    const request = requests[requestIndex]
    requestIndex += 1
    return request?.promise ?? Promise.resolve()
  })

  return { refetch, requests }
}

describe("useScanHistoryRefresh", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setVisibilityState("visible")
    scanHooksMocks.useScanStatistics.mockReturnValue({
      data: { pending: 0, running: 0 },
    })
  })

  afterEach(() => {
    setVisibilityState("visible")
    vi.useRealTimers()
  })

  it("concurrently refetches active statistics and list queries", async () => {
    const completedAt = new Date("2026-07-25T08:09:10.000Z")
    vi.useFakeTimers()
    vi.setSystemTime(completedAt)
    const { queryClient, result } = renderHookWithProviders(() => useScanHistoryRefresh())
    const { refetch, requests } = mockRefetches(queryClient)
    let refreshPromise!: Promise<void>

    act(() => {
      refreshPromise = result.current.refresh()
    })

    expect(result.current.isRefreshing).toBe(true)
    expect(result.current.lastRefreshedAt).toEqual(completedAt)
    expect(refetch.mock.calls.map(([filters]) => filters)).toEqual([
      { exact: true, queryKey: scanKeys.statistics(), type: "active" },
      { queryKey: scanKeys.lists(), type: "active" },
    ])

    await act(async () => {
      requests.forEach((request) => request.resolve())
      await refreshPromise
    })

    expect(result.current.isRefreshing).toBe(false)
    expect(result.current.lastRefreshedAt).toEqual(completedAt)
  })

  it("prevents a second coordinated refresh while the first one is pending", async () => {
    const { queryClient, result } = renderHookWithProviders(() => useScanHistoryRefresh())
    const { refetch, requests } = mockRefetches(queryClient)
    let refreshPromise!: Promise<void>

    act(() => {
      refreshPromise = result.current.refresh()
      void result.current.refresh()
    })

    expect(refetch).toHaveBeenCalledTimes(2)

    await act(async () => {
      requests.forEach((request) => request.resolve())
      await refreshPromise
    })
  })

  it("polls active scans and stops after the statistics become terminal", async () => {
    vi.useFakeTimers()
    scanHooksMocks.useScanStatistics.mockReturnValue({
      data: { pending: 1, running: 0 },
    })
    const { queryClient, rerender } = renderHookWithProviders(() => useScanHistoryRefresh())
    const { refetch, requests } = mockRefetches(queryClient)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(SCAN_HISTORY_AUTO_REFRESH_MS)
    })

    expect(refetch).toHaveBeenCalledTimes(2)

    await act(async () => {
      requests.forEach((request) => request.resolve())
      await Promise.resolve()
    })

    scanHooksMocks.useScanStatistics.mockReturnValue({
      data: { pending: 0, running: 0 },
    })
    rerender()

    await act(async () => {
      await vi.advanceTimersByTimeAsync(SCAN_HISTORY_AUTO_REFRESH_MS)
    })

    expect(refetch).toHaveBeenCalledTimes(2)
  })

  it("pauses while hidden and refreshes immediately when visible again", async () => {
    vi.useFakeTimers()
    scanHooksMocks.useScanStatistics.mockReturnValue({
      data: { pending: 0, running: 1 },
    })
    const { queryClient } = renderHookWithProviders(() => useScanHistoryRefresh())
    const { refetch, requests } = mockRefetches(queryClient)

    act(() => {
      setVisibilityState("hidden")
      document.dispatchEvent(new Event("visibilitychange"))
    })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(SCAN_HISTORY_AUTO_REFRESH_MS * 2)
    })
    expect(refetch).not.toHaveBeenCalled()

    act(() => {
      setVisibilityState("visible")
      document.dispatchEvent(new Event("visibilitychange"))
    })

    expect(refetch).toHaveBeenCalledTimes(2)

    await act(async () => {
      requests.forEach((request) => request.resolve())
      await Promise.resolve()
    })
  })

  it("does not overlap automatic refresh batches or manual refresh", async () => {
    vi.useFakeTimers()
    scanHooksMocks.useScanStatistics.mockReturnValue({
      data: { pending: 1, running: 0 },
    })
    const { queryClient, result } = renderHookWithProviders(() => useScanHistoryRefresh())
    const { refetch, requests } = mockRefetches(queryClient)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(SCAN_HISTORY_AUTO_REFRESH_MS * 2)
    })
    expect(refetch).toHaveBeenCalledTimes(2)

    await act(async () => {
      await result.current.refresh()
    })
    expect(refetch).toHaveBeenCalledTimes(2)

    await act(async () => {
      requests.forEach((request) => request.resolve())
      await Promise.resolve()
    })
  })
})

function setVisibilityState(state: "hidden" | "visible") {
  Object.defineProperty(document, "visibilityState", {
    configurable: true,
    value: state,
  })
}
