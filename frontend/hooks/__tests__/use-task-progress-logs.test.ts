import { act } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { useTaskProgressLogs } from "@/hooks/use-task-progress-logs"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import type { GetScanLogsResponse } from "@/types/scan.types"

const serviceMocks = vi.hoisted(() => ({
  getScanLogs: vi.fn(),
}))

vi.mock("@/services/scan.service", () => ({
  getScanLogs: serviceMocks.getScanLogs,
}))

describe("useTaskProgressLogs", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  async function flushEffects() {
    await act(async () => {
      await Promise.resolve()
      await Promise.resolve()
    })
  }

  it("drains historical pages and polls from the last tail token", async () => {
    serviceMocks.getScanLogs
      .mockResolvedValueOnce({
        results: [{ id: 1, taskId: 7001, level: "info", content: "started", createdAt: "2026-06-18T00:00:00Z" }],
        nextPageToken: "cursor-1",
      } satisfies GetScanLogsResponse)
      .mockResolvedValueOnce({
        results: [{ id: 2, taskId: 7002, level: "info", content: "done", createdAt: "2026-06-18T00:00:01Z" }],
      } satisfies GetScanLogsResponse)
      .mockResolvedValueOnce({
        results: [
          { id: 2, taskId: 7002, level: "info", content: "done", createdAt: "2026-06-18T00:00:01Z" },
          { id: 3, taskId: 7002, level: "info", content: "[server] committed", createdAt: "2026-06-18T00:00:00Z" },
        ],
      } satisfies GetScanLogsResponse)

    const { result, unmount } = renderHookWithProviders(() =>
      useTaskProgressLogs({ scanId: 7, pollingInterval: 1000 })
    )

    await flushEffects()

    expect(serviceMocks.getScanLogs).toHaveBeenNthCalledWith(1, 7, { pageSize: 200 })
    expect(serviceMocks.getScanLogs).toHaveBeenNthCalledWith(2, 7, { pageSize: 200, pageToken: "cursor-1" })
    expect(result.current.logs.map(log => log.id)).toEqual([1, 2])

    await act(async () => {
      vi.advanceTimersByTime(1000)
      await Promise.resolve()
      await Promise.resolve()
    })

    expect(serviceMocks.getScanLogs).toHaveBeenNthCalledWith(3, 7, {
      pageSize: 200,
      pageToken: "cursor-1",
    })
    expect(result.current.logs.map(log => log.id)).toEqual([1, 2, 3])

    unmount()
  })

  it("keeps polling without a next page token and de-duplicates repeated log ids", async () => {
    serviceMocks.getScanLogs
      .mockResolvedValueOnce({
        results: [{ id: 1, taskId: 7001, level: "info", content: "started", createdAt: "2026-06-18T00:00:00Z" }],
      } satisfies GetScanLogsResponse)
      .mockResolvedValueOnce({
        results: [
          { id: 1, taskId: 7001, level: "info", content: "started again", createdAt: "2026-06-18T00:00:00Z" },
          { id: 2, taskId: 7002, level: "info", content: "finished", createdAt: "2026-06-18T00:00:01Z" },
        ],
      } satisfies GetScanLogsResponse)

    const { result, unmount } = renderHookWithProviders(() =>
      useTaskProgressLogs({ scanId: 7, pollingInterval: 1000 })
    )

    await flushEffects()

    await act(async () => {
      vi.advanceTimersByTime(1000)
      await Promise.resolve()
      await Promise.resolve()
    })

    expect(serviceMocks.getScanLogs).toHaveBeenNthCalledWith(2, 7, { pageSize: 200 })
    expect(result.current.logs.map((log) => log.id)).toEqual([1, 2])
    expect(result.current.logs.map((log) => log.content)).toEqual(["started", "finished"])

    unmount()
  })

  it("keeps the log array reference when an incremental poll has no new rows", async () => {
    const response = {
      results: [{ id: 1, taskId: 7001, level: "info", content: "started", createdAt: "2026-06-18T00:00:00Z" }],
    } satisfies GetScanLogsResponse
    serviceMocks.getScanLogs.mockResolvedValue(response)

    const { result, unmount } = renderHookWithProviders(() =>
      useTaskProgressLogs({ scanId: 7, pollingInterval: 1000 })
    )

    await flushEffects()
    const initialLogs = result.current.logs

    await act(async () => {
      vi.advanceTimersByTime(1000)
      await Promise.resolve()
      await Promise.resolve()
    })

    expect(result.current.logs).toBe(initialLogs)
    unmount()
  })

  it("loads every page for a terminal scan with polling disabled", async () => {
    serviceMocks.getScanLogs
      .mockResolvedValueOnce({
        results: [{ id: 1, taskId: 7001, level: "info", content: "[agent] started", createdAt: "2026-06-18T00:00:00Z" }],
        nextPageToken: "opaque-tail",
      } satisfies GetScanLogsResponse)
      .mockResolvedValueOnce({
        results: [{ id: 2, taskId: 7001, level: "info", content: "[engine/stdout] completed", createdAt: "2026-06-18T00:00:01Z" }],
      } satisfies GetScanLogsResponse)
    const { result, unmount } = renderHookWithProviders(() => useTaskProgressLogs({ scanId: 7, pollingInterval: 0, maxLogs: 1 }))
    await flushEffects()
    expect(serviceMocks.getScanLogs).toHaveBeenCalledTimes(2)
    expect(result.current.logs.map(log => log.id)).toEqual([2])
    expect(result.current.loading).toBe(false)
    unmount()
  })

  it("fetches terminal logs after an older in-flight poll finishes", async () => {
    let resolvePoll: (value: GetScanLogsResponse) => void = () => {}
    const initial: GetScanLogsResponse = { results: [{ id: 1, taskId: 7001, level: "info", content: "started", createdAt: "2026-06-18T00:00:00Z" }] }
    serviceMocks.getScanLogs
      .mockResolvedValueOnce(initial)
      .mockImplementationOnce(() => new Promise<GetScanLogsResponse>(resolve => { resolvePoll = resolve }))
      .mockResolvedValueOnce({ results: [...initial.results, { id: 2, taskId: 7001, level: "info", content: "[server] terminal committed", createdAt: "2026-06-18T00:00:01Z" }] } satisfies GetScanLogsResponse)
    const { result, rerender, unmount } = renderHookWithProviders(
      ({ interval }: { interval: number }) => useTaskProgressLogs({ scanId: 7, pollingInterval: interval }),
      { initialProps: { interval: 1000 } },
    )
    await flushEffects()
    await act(async () => { vi.advanceTimersByTime(1000) })
    expect(serviceMocks.getScanLogs).toHaveBeenCalledTimes(2)
    rerender({ interval: 0 })
    expect(serviceMocks.getScanLogs).toHaveBeenCalledTimes(2)
    await act(async () => { resolvePoll(initial) })
    await flushEffects()
    expect(serviceMocks.getScanLogs).toHaveBeenCalledTimes(3)
    expect(result.current.logs.map(log => log.id)).toEqual([1, 2])
    unmount()
  })

  it("does not reinsert evicted IDs when polling an unchanged tail", async () => {
    serviceMocks.getScanLogs.mockResolvedValue({
      results: [
        { id: 1, taskId: 7001, level: "info", content: "old", createdAt: "2026-06-18T00:00:00Z" },
        { id: 2, taskId: 7001, level: "info", content: "latest", createdAt: "2026-06-18T00:00:01Z" },
      ],
    } satisfies GetScanLogsResponse)
    const { result, unmount } = renderHookWithProviders(() => useTaskProgressLogs({ scanId: 7, pollingInterval: 1000, maxLogs: 1 }))
    await flushEffects()
    await act(async () => { vi.advanceTimersByTime(1000) })
    await flushEffects()
    expect(result.current.logs.map(log => log.id)).toEqual([2])
    unmount()
  })

  it("ignores responses from the previous scan and prevents overlapping polls", async () => {
    let resolveOld: (value: GetScanLogsResponse) => void = () => {}
    serviceMocks.getScanLogs
      .mockImplementationOnce(() => new Promise<GetScanLogsResponse>(resolve => { resolveOld = resolve }))
      .mockResolvedValue({ results: [{ id: 2, taskId: 8001, level: "info", content: "new scan", createdAt: "2026-06-18T00:00:00Z" }] })
    const { result, rerender, unmount } = renderHookWithProviders(
      ({ scanId }: { scanId: number }) => useTaskProgressLogs({ scanId, pollingInterval: 1000 }),
      { initialProps: { scanId: 7 } },
    )
    await act(async () => { vi.advanceTimersByTime(2000) })
    expect(serviceMocks.getScanLogs).toHaveBeenCalledTimes(1)
    rerender({ scanId: 8 })
    await flushEffects()
    await act(async () => {
      resolveOld({ results: [{ id: 1, taskId: 7001, level: "info", content: "old scan", createdAt: "2026-06-18T00:00:00Z" }] })
    })
    expect(result.current.logs.map(log => log.id)).toEqual([2])
    unmount()
  })
})
