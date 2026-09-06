import { act, renderHook } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { useAgentManagementRefresh } from "@/hooks/use-agent-management-refresh"

function deferred() {
  let resolve!: () => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<void>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })

  return { promise, reject, resolve }
}

describe("useAgentManagementRefresh", () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it("waits for all visible content requests to settle before recording completion", async () => {
    const mountedAt = new Date("2026-07-25T08:08:09.000Z")
    const completedAt = new Date("2026-07-25T08:09:10.000Z")
    vi.useFakeTimers()
    vi.setSystemTime(mountedAt)
    const current = deferred()
    const summary = deferred()
    const filterOptions = deferred()
    const refetchCurrent = vi.fn(() => current.promise)
    const refetchSummary = vi.fn(() => summary.promise)
    const refetchFilterOptions = vi.fn(() => filterOptions.promise)
    const { result } = renderHook(() => useAgentManagementRefresh({
      refetchCurrent,
      refetchSummary,
      refetchFilterOptions,
    }))
    let refreshPromise!: Promise<void>

    act(() => {
      refreshPromise = result.current.refresh()
    })

    expect(result.current.isRefreshing).toBe(true)
    expect(refetchCurrent).toHaveBeenCalledTimes(1)
    expect(refetchSummary).toHaveBeenCalledTimes(1)
    expect(refetchFilterOptions).toHaveBeenCalledTimes(1)
    expect(result.current.lastRefreshedAt).toEqual(mountedAt)

    await act(async () => {
      current.reject(new Error("current list failed"))
      summary.resolve()
      await Promise.resolve()
    })

    expect(result.current.isRefreshing).toBe(true)
    expect(result.current.lastRefreshedAt).toEqual(mountedAt)

    await act(async () => {
      vi.setSystemTime(completedAt)
      filterOptions.resolve()
      await refreshPromise
    })

    expect(result.current.isRefreshing).toBe(false)
    expect(result.current.lastRefreshedAt).toEqual(completedAt)
  })

  it("rejects duplicate coordinated refreshes while one is pending", async () => {
    const current = deferred()
    const summary = deferred()
    const filterOptions = deferred()
    const refetchCurrent = vi.fn(() => current.promise)
    const refetchSummary = vi.fn(() => summary.promise)
    const refetchFilterOptions = vi.fn(() => filterOptions.promise)
    const { result } = renderHook(() => useAgentManagementRefresh({
      refetchCurrent,
      refetchSummary,
      refetchFilterOptions,
    }))
    let refreshPromise!: Promise<void>

    act(() => {
      refreshPromise = result.current.refresh()
      void result.current.refresh()
    })

    expect(refetchCurrent).toHaveBeenCalledTimes(1)
    expect(refetchSummary).toHaveBeenCalledTimes(1)
    expect(refetchFilterOptions).toHaveBeenCalledTimes(1)

    await act(async () => {
      current.resolve()
      summary.resolve()
      filterOptions.resolve()
      await refreshPromise
    })
  })
})
