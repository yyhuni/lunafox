import { act } from "@testing-library/react"
import type { QueryClient } from "@tanstack/react-query"
import { afterEach, describe, expect, it, vi } from "vitest"

import { overviewRefreshQueryKeys, useOverviewRefresh } from "@/hooks/use-overview-refresh"
import { agentKeys } from "@/hooks/use-agents"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"

function deferred() {
  let resolve!: () => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<void>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })

  return { promise, reject, resolve }
}

function mockRefetches(queryClient: QueryClient) {
  const requests = overviewRefreshQueryKeys.map(() => deferred())
  let requestIndex = 0
  const refetch = vi.spyOn(queryClient, "refetchQueries").mockImplementation(() => {
    const request = requests[requestIndex]
    requestIndex += 1
    return request?.promise ?? Promise.resolve()
  })

  return { refetch, requests }
}

describe("useOverviewRefresh", () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it("concurrently refetches the exact eight active Overview query scopes", async () => {
    const { queryClient, result } = renderHookWithProviders(() => useOverviewRefresh())
    const { refetch, requests } = mockRefetches(queryClient)
    let refreshPromise!: Promise<void>

    act(() => {
      refreshPromise = result.current.refresh()
    })

    expect(result.current.isRefreshing).toBe(true)
    expect(refetch).toHaveBeenCalledTimes(overviewRefreshQueryKeys.length)
    expect(overviewRefreshQueryKeys).toContainEqual(agentKeys.clusterSummary())
    expect(overviewRefreshQueryKeys).toContainEqual(agentKeys.locationMap())
    expect(refetch.mock.calls.map(([filters]) => filters)).toEqual(
      overviewRefreshQueryKeys.map((queryKey) => ({ exact: true, queryKey, type: "active" }))
    )

    await act(async () => {
      requests.forEach((request) => request.resolve())
      await refreshPromise
    })

    expect(result.current.isRefreshing).toBe(false)
  })

  it("waits for every request, records completion, and settles after a partial failure", async () => {
    const mountedAt = new Date("2026-07-24T08:08:09.000Z")
    const completedAt = new Date("2026-07-24T08:09:10.000Z")
    vi.useFakeTimers()
    vi.setSystemTime(mountedAt)
    const { queryClient, result } = renderHookWithProviders(() => useOverviewRefresh())
    const { requests } = mockRefetches(queryClient)
    let refreshPromise!: Promise<void>

    expect(result.current.lastRefreshedAt).toEqual(mountedAt)

    act(() => {
      refreshPromise = result.current.refresh()
    })

    await act(async () => {
      requests[0]?.reject(new Error("asset statistics unavailable"))
      requests.slice(1, -1).forEach((request) => request.resolve())
      await Promise.resolve()
    })

    expect(result.current.isRefreshing).toBe(true)
    expect(result.current.lastRefreshedAt).toEqual(mountedAt)

    await act(async () => {
      vi.setSystemTime(completedAt)
      requests.at(-1)?.resolve()
      await refreshPromise
    })

    expect(result.current.isRefreshing).toBe(false)
    expect(result.current.lastRefreshedAt).toEqual(completedAt)
  })

  it("prevents a second coordinated refresh while the first one is pending", async () => {
    const { queryClient, result } = renderHookWithProviders(() => useOverviewRefresh())
    const { refetch, requests } = mockRefetches(queryClient)
    let firstRefresh!: Promise<void>

    act(() => {
      firstRefresh = result.current.refresh()
      void result.current.refresh()
    })

    expect(refetch).toHaveBeenCalledTimes(overviewRefreshQueryKeys.length)

    await act(async () => {
      requests.forEach((request) => request.resolve())
      await firstRefresh
    })
  })
})
