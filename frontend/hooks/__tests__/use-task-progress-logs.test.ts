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

  it("polls task progress logs with opaque AIP page tokens", async () => {
    serviceMocks.getScanLogs
      .mockResolvedValueOnce({
        results: [{ id: 1, taskId: 7001, level: "info", content: "started", createdAt: "2026-06-18T00:00:00Z" }],
        nextPageToken: "cursor-1",
      } satisfies GetScanLogsResponse)
      .mockResolvedValueOnce({
        results: [{ id: 2, taskId: 7002, level: "info", content: "done", createdAt: "2026-06-18T00:00:01Z" }],
        nextPageToken: "cursor-2",
      } satisfies GetScanLogsResponse)

    const { unmount } = renderHookWithProviders(() =>
      useTaskProgressLogs({ scanId: 7, pollingInterval: 1000 })
    )

    await flushEffects()

    expect(serviceMocks.getScanLogs).toHaveBeenNthCalledWith(1, 7, { pageSize: 200 })

    await act(async () => {
      vi.advanceTimersByTime(1000)
      await Promise.resolve()
      await Promise.resolve()
    })

    expect(serviceMocks.getScanLogs).toHaveBeenNthCalledWith(2, 7, {
      pageSize: 200,
      pageToken: "cursor-1",
    })

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
})
