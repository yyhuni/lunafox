import { act, waitFor } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import {
  SCAN_AGENT_PICKER_PAGE_SIZE,
  SCAN_AGENT_PICKER_REFRESH_MS,
  SCAN_AGENT_PICKER_SEARCH_DEBOUNCE_MS,
  useScanAgentPicker,
} from "@/hooks/use-scan-agent-picker"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import type { Agent, AgentsResponse } from "@/types/agent.types"

const agentServiceMocks = vi.hoisted(() => ({
  getAgents: vi.fn(),
}))

vi.mock("@/services/agent.service", () => ({
  agentService: agentServiceMocks,
}))

function agent(id: number, displayName = `agent-${id}`): Agent {
  return {
    id,
    name: displayName,
    resourceName: `agents/${id}`,
    displayName,
    status: "online",
    maxTasks: 4,
    cpuThreshold: 80,
    memThreshold: 80,
    diskThreshold: 90,
    health: { state: "healthy" },
    createdAt: "2026-08-04T00:00:00Z",
  }
}

function page(results: Agent[], nextPageToken?: string): AgentsResponse {
  return {
    results,
    totalSize: results.length,
    pageSize: SCAN_AGENT_PICKER_PAGE_SIZE,
    ...(nextPageToken ? { nextPageToken } : {}),
  }
}

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, reject, resolve }
}

describe("useScanAgentPicker", () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it("keeps the canonical collection closed and opens with one bounded first page", async () => {
    agentServiceMocks.getAgents.mockResolvedValue(page([agent(1)]))
    const { result, rerender } = renderHookWithProviders(
      ({ open }) => useScanAgentPicker({ open, search: "" }),
      { initialProps: { open: false } },
    )

    expect(agentServiceMocks.getAgents).not.toHaveBeenCalled()

    rerender({ open: true })
    await waitFor(() => expect(result.current.agents).toHaveLength(1))

    expect(agentServiceMocks.getAgents).toHaveBeenCalledWith(
      {
        pageSize: SCAN_AGENT_PICKER_PAGE_SIZE,
        orderBy: "createdAt desc",
      },
      expect.any(AbortSignal),
    )
    expect(JSON.stringify(agentServiceMocks.getAgents.mock.calls)).not.toContain("1000")
  })

  it("debounces normalized server search and isolates a late older generation", async () => {
    const oldSearch = deferred<AgentsResponse>()
    const currentSearch = deferred<AgentsResponse>()
    agentServiceMocks.getAgents.mockImplementation((params: { filter?: string }) => {
      if (params.filter?.includes("old")) return oldSearch.promise
      if (params.filter?.includes("current query")) return currentSearch.promise
      return Promise.resolve(page([]))
    })

    const { result, rerender } = renderHookWithProviders(
      ({ search }) => useScanAgentPicker({ open: true, search }),
      { initialProps: { search: "old" } },
    )
    await waitFor(() => expect(agentServiceMocks.getAgents).toHaveBeenCalledTimes(1))

    rerender({ search: "  current   query  " })
    await vi.advanceTimersByTimeAsync(SCAN_AGENT_PICKER_SEARCH_DEBOUNCE_MS - 1)
    expect(agentServiceMocks.getAgents).toHaveBeenCalledTimes(1)

    await vi.advanceTimersByTimeAsync(1)
    await waitFor(() => expect(agentServiceMocks.getAgents).toHaveBeenCalledTimes(2))
    expect(agentServiceMocks.getAgents.mock.calls[1]?.[0]).toMatchObject({
      filter: '(displayName="current query" || observedHostname="current query" || connectionIp="current query")',
    })

    currentSearch.resolve(page([agent(2, "current")]))
    await waitFor(() => expect(result.current.agents.map((item) => item.id)).toEqual([2]))
    oldSearch.resolve(page([agent(1, "old")]))
    await act(async () => undefined)
    expect(result.current.agents.map((item) => item.id)).toEqual([2])
  })

  it("uses sequential opaque tokens, canonical dedupe, and one in-flight next-page request", async () => {
    const secondPage = deferred<AgentsResponse>()
    agentServiceMocks.getAgents.mockImplementation((params: { pageToken?: string }) => {
      if (params.pageToken === "opaque-2") return secondPage.promise
      if (params.pageToken === "opaque-3") return Promise.resolve(page([agent(3)]))
      return Promise.resolve(page([agent(1), agent(2)], "opaque-2"))
    })

    const { result } = renderHookWithProviders(() => useScanAgentPicker({ open: true, search: "" }))
    await waitFor(() => expect(result.current.hasNextPage).toBe(true))

    act(() => {
      void result.current.loadNextPage()
      void result.current.loadNextPage()
    })
    await waitFor(() => {
      expect(agentServiceMocks.getAgents.mock.calls.filter(([params]) => params.pageToken === "opaque-2")).toHaveLength(1)
    })

    secondPage.resolve(page([agent(2, "duplicate"), agent(3)], "opaque-3"))
    await waitFor(() => expect(result.current.agents.map((item) => item.id)).toEqual([1, 2, 3]))

    await act(async () => {
      await result.current.loadNextPage()
    })
    expect(agentServiceMocks.getAgents.mock.calls.at(-1)?.[0]).toMatchObject({ pageToken: "opaque-3" })
  })

  it("retains rows and the failed token until explicit next-page recovery", async () => {
    agentServiceMocks.getAgents
      .mockResolvedValueOnce(page([agent(1)], "retry-token"))
      .mockRejectedValueOnce(new Error("page unavailable"))
      .mockResolvedValueOnce(page([agent(2)]))

    const { result } = renderHookWithProviders(() => useScanAgentPicker({ open: true, search: "" }))
    await waitFor(() => expect(result.current.hasNextPage).toBe(true))

    await act(async () => {
      await result.current.loadNextPage()
    })
    await waitFor(() => expect(result.current.isNextPageError).toBe(true))
    expect(result.current.agents.map((item) => item.id)).toEqual([1])

    await act(async () => {
      await result.current.loadNextPage()
    })
    expect(agentServiceMocks.getAgents).toHaveBeenCalledTimes(2)

    await act(async () => {
      await result.current.retryNextPage()
    })
    await waitFor(() => expect(result.current.agents.map((item) => item.id)).toEqual([1, 2]))
    expect(agentServiceMocks.getAgents.mock.calls.at(-1)?.[0]).toMatchObject({ pageToken: "retry-token" })
    expect(result.current.isNextPageError).toBe(false)
  })

  it("refreshes only loaded pages while open and cancels the active generation on close", async () => {
    const pendingRefresh = deferred<AgentsResponse>()
    let refreshStarted = false
    agentServiceMocks.getAgents.mockImplementation((params: { pageToken?: string }) => {
      if (refreshStarted && !params.pageToken) return pendingRefresh.promise
      if (params.pageToken === "loaded-page-2") return Promise.resolve(page([agent(2)]))
      return Promise.resolve(page([agent(1)], "loaded-page-2"))
    })

    const { result, rerender } = renderHookWithProviders(
      ({ open }) => useScanAgentPicker({ open, search: "" }),
      { initialProps: { open: true } },
    )
    await waitFor(() => expect(result.current.hasNextPage).toBe(true))
    await act(async () => {
      await result.current.loadNextPage()
    })

    refreshStarted = true
    await vi.advanceTimersByTimeAsync(SCAN_AGENT_PICKER_REFRESH_MS)
    await waitFor(() => expect(agentServiceMocks.getAgents).toHaveBeenCalledTimes(3))
    const refreshSignal = agentServiceMocks.getAgents.mock.calls[2]?.[1] as AbortSignal

    rerender({ open: false })
    await waitFor(() => expect(refreshSignal.aborted).toBe(true))
    await vi.advanceTimersByTimeAsync(SCAN_AGENT_PICKER_REFRESH_MS * 2)
    expect(agentServiceMocks.getAgents).toHaveBeenCalledTimes(3)
  })

  it("refreshes every loaded page without extending the loaded range", async () => {
    agentServiceMocks.getAgents.mockImplementation((params: { pageToken?: string }) => {
      if (params.pageToken === "loaded-page-2") return Promise.resolve(page([agent(2)]))
      return Promise.resolve(page([agent(1)], "loaded-page-2"))
    })

    const { result } = renderHookWithProviders(() => useScanAgentPicker({ open: true, search: "" }))
    await waitFor(() => expect(result.current.hasNextPage).toBe(true))
    await act(async () => {
      await result.current.loadNextPage()
    })

    await vi.advanceTimersByTimeAsync(SCAN_AGENT_PICKER_REFRESH_MS)
    await waitFor(() => expect(agentServiceMocks.getAgents).toHaveBeenCalledTimes(4))
    expect(agentServiceMocks.getAgents.mock.calls.slice(2).map(([params]) => params.pageToken)).toEqual([
      undefined,
      "loaded-page-2",
    ])
  })

  it("retains loaded rows after refresh failure and clears stale state after recovery", async () => {
    agentServiceMocks.getAgents
      .mockResolvedValueOnce(page([agent(1, "before-refresh")]))
      .mockRejectedValueOnce(new Error("refresh unavailable"))
      .mockResolvedValueOnce(page([agent(1, "after-recovery")]))

    const { result } = renderHookWithProviders(() => useScanAgentPicker({ open: true, search: "" }))
    await waitFor(() => expect(result.current.agents[0]?.displayName).toBe("before-refresh"))

    await vi.advanceTimersByTimeAsync(SCAN_AGENT_PICKER_REFRESH_MS)
    await waitFor(() => expect(result.current.isRefreshError).toBe(true))
    expect(result.current.agents[0]?.displayName).toBe("before-refresh")
    expect(result.current.loadedPageCount).toBe(1)

    await act(async () => {
      await result.current.retryCurrentGeneration()
    })
    await waitFor(() => expect(result.current.agents[0]?.displayName).toBe("after-recovery"))
    expect(result.current.isRefreshError).toBe(false)
  })

  it("blank search starts a fresh default generation without carrying a page token", async () => {
    agentServiceMocks.getAgents.mockImplementation((params: { filter?: string; pageToken?: string }) => {
      if (params.filter) return Promise.resolve(page([agent(1)], "search-page-2"))
      return Promise.resolve(page([agent(2)]))
    })

    const { result, rerender } = renderHookWithProviders(
      ({ search }) => useScanAgentPicker({ open: true, search }),
      { initialProps: { search: "edge" } },
    )
    await waitFor(() => expect(result.current.agents.map((item) => item.id)).toEqual([1]))

    rerender({ search: "   " })
    await vi.advanceTimersByTimeAsync(SCAN_AGENT_PICKER_SEARCH_DEBOUNCE_MS)
    await waitFor(() => expect(result.current.agents.map((item) => item.id)).toEqual([2]))

    expect(agentServiceMocks.getAgents.mock.calls.at(-1)?.[0]).toEqual({
      pageSize: SCAN_AGENT_PICKER_PAGE_SIZE,
      orderBy: "createdAt desc",
    })
  })

  it("clears the debounced search generation while closed before a rapid reopen", async () => {
    agentServiceMocks.getAgents.mockImplementation((params: { filter?: string }) => (
      Promise.resolve(params.filter ? page([agent(1, "filtered")]) : page([agent(2, "default")]))
    ))

    const { result, rerender } = renderHookWithProviders(
      ({ open, search }) => useScanAgentPicker({ open, search }),
      { initialProps: { open: true, search: "edge" } },
    )
    await waitFor(() => expect(result.current.agents[0]?.displayName).toBe("filtered"))

    rerender({ open: false, search: "" })
    await waitFor(() => expect(result.current.normalizedSearch).toBe(""))
    rerender({ open: true, search: "" })

    await waitFor(() => expect(result.current.agents[0]?.displayName).toBe("default"))
    expect(agentServiceMocks.getAgents.mock.calls.at(-1)?.[0]).toEqual({
      pageSize: SCAN_AGENT_PICKER_PAGE_SIZE,
      orderBy: "createdAt desc",
    })
  })
})
