import * as React from "react"
import { act, cleanup, renderHook } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

import { type FetchLogStreamParams, useLogStreamViewer } from "@/hooks/use-log-stream-viewer"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"

const line = (id: string) => ({ id, ts: id, stream: "stdout", line: id, truncated: false })
const source = readFileSync(path.resolve(process.cwd(), "hooks/use-log-stream-viewer.ts"), "utf8")
const supportedWindowSizes = [100, 200, 500, 1000, 2000, 5000] as const

function logIds(start: number, end: number) {
  return Array.from({ length: end - start + 1 }, (_, index) => `l${start + index}`)
}

const response = (ids: string[], overrides = {}) => ({
  logs: ids.map(line),
  nextCursor: "follow-1",
  previousCursor: "",
  hasOlder: false,
  hasNewer: false,
  caughtUp: true,
  gap: false,
  gapReason: "",
  ...overrides,
})

async function flushMicrotasks() {
  await Promise.resolve()
  await Promise.resolve()
}

async function waitForCondition(predicate: () => boolean, attempts = 20) {
  for (let index = 0; index < attempts; index += 1) {
    if (predicate()) return
    await act(async () => {
      await flushMicrotasks()
    })
  }
  expect(predicate()).toBe(true)
}

function deferred<T>() {
  let resolve: (value: T) => void
  let reject: (reason?: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve: resolve!, reject: reject! }
}

describe("useLogStreamViewer", () => {
  afterEach(() => {
    cleanup()
    vi.clearAllTimers()
    vi.useRealTimers()
  })

  it("hydrates history before immediately catching up from the first follow cursor", async () => {
    const fetchLogs = vi
      .fn()
      .mockResolvedValueOnce(response(["l2"], { nextCursor: "follow-1", previousCursor: "older-1", hasOlder: true }))
      .mockResolvedValueOnce(response(["l1"], { nextCursor: "must-not-replace-follow", previousCursor: "", hasOlder: false }))
      .mockResolvedValueOnce(response(["l3"], { nextCursor: "follow-2", hasNewer: false, caughtUp: true }))

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, windowSize: 2 }),
    )

    await waitForCondition(() => result.current.phase === "following")
    expect(result.current.lines.map((item) => item.id)).toEqual(["l2", "l3"])
    expect(fetchLogs).toHaveBeenNthCalledWith(2, expect.objectContaining({ cursor: "older-1", direction: "older", limit: 500 }))
    expect(fetchLogs).toHaveBeenNthCalledWith(3, expect.objectContaining({ cursor: "follow-1", direction: "newer", limit: 500 }))
    unmount()
  })

  it("hydrates while auto refresh is disabled but does not start follow", async () => {
    const fetchLogs = vi
      .fn()
      .mockResolvedValueOnce(response(["l2"], { previousCursor: "older-1", hasOlder: true }))
      .mockResolvedValueOnce(response(["l1"], { previousCursor: "", hasOlder: false }))

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, windowSize: 2, pollingEnabled: false }),
    )

    await waitForCondition(() => result.current.phase === "ready")
    expect(result.current.lines.map((item) => item.id)).toEqual(["l1", "l2"])
    expect(fetchLogs).toHaveBeenCalledTimes(2)
    unmount()
  })

  it("waits for an in-flight initial latest request before re-enabling follow", async () => {
    const initialLatest = deferred<ReturnType<typeof response>>()
    const fetchLogs = vi
      .fn()
      .mockImplementationOnce(() => initialLatest.promise)
      .mockResolvedValueOnce(response(["l2"], { nextCursor: "follow-2", hasNewer: false, caughtUp: true }))

    const { result, rerender, unmount } = renderHookWithProviders(
      ({ pollingEnabled }) => useLogStreamViewer({ enabled: true, fetchLogs, windowSize: 2, pollingEnabled }),
      { initialProps: { pollingEnabled: false } },
    )
    await waitForCondition(() => fetchLogs.mock.calls.length === 1)

    await act(async () => {
      rerender({ pollingEnabled: true })
      await flushMicrotasks()
    })
    expect(fetchLogs).toHaveBeenCalledTimes(1)

    await act(async () => {
      initialLatest.resolve(response(["l1"], { nextCursor: "follow-1", hasNewer: true, caughtUp: false }))
      await flushMicrotasks()
    })
    await waitForCondition(() => fetchLogs.mock.calls.length === 2 && result.current.phase === "following")
    expect(fetchLogs).toHaveBeenNthCalledWith(2, expect.objectContaining({
      cursor: "follow-1",
      direction: "newer",
      limit: 500,
    }))
    expect(result.current.lines.map((item) => item.id)).toEqual(["l1", "l2"])
    unmount()
  })

  it("does not start a second latest request while shrinking during initial loading", async () => {
    vi.useFakeTimers()
    const initialLatest = deferred<ReturnType<typeof response>>()
    const fetchLogs = vi
      .fn()
      .mockImplementationOnce(() => initialLatest.promise)
      .mockResolvedValueOnce(response(["l4"], { nextCursor: "follow-2", hasNewer: false, caughtUp: true }))

    const { result, rerender, unmount } = renderHookWithProviders(
      ({ windowSize }) => useLogStreamViewer({ enabled: true, fetchLogs, windowSize, pollIntervalMs: 50 }),
      { initialProps: { windowSize: 3 } },
    )
    await waitForCondition(() => fetchLogs.mock.calls.length === 1)

    await act(async () => {
      rerender({ windowSize: 2 })
      await flushMicrotasks()
    })
    expect(fetchLogs).toHaveBeenCalledTimes(1)

    await act(async () => {
      initialLatest.resolve(response(["l1", "l2", "l3"], { nextCursor: "follow-1", hasNewer: false, caughtUp: true }))
      await flushMicrotasks()
    })
    await waitForCondition(() => result.current.phase === "following")
    expect(result.current.lines.map((item) => item.id)).toEqual(["l2", "l3"])
    expect(fetchLogs).toHaveBeenCalledTimes(1)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(50)
    })
    await waitForCondition(() => fetchLogs.mock.calls.length === 2)
    expect(fetchLogs).toHaveBeenNthCalledWith(2, expect.objectContaining({
      cursor: "follow-1",
      direction: "newer",
      limit: 500,
    }))
    unmount()
  })

  it.each(supportedWindowSizes)("builds the latest %i-line window in 500-row transport batches", async (windowSize) => {
    let olderBatch = 0
    const fetchLogs = vi.fn((params: FetchLogStreamParams) => {
      if (params.direction === "older") {
        olderBatch += 1
        const end = windowSize - olderBatch * 500
        const start = Math.max(1, end - 499)
        return Promise.resolve(response(logIds(start, end), {
          previousCursor: start > 1 ? `older-${olderBatch + 1}` : "",
          hasOlder: start > 1,
        }))
      }

      const start = Math.max(1, windowSize - 499)
      return Promise.resolve(response(logIds(start, windowSize), {
        previousCursor: start > 1 ? "older-1" : "",
        hasOlder: start > 1,
      }))
    })

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, windowSize, pollingEnabled: false }),
    )

    await waitForCondition(() => result.current.phase === "ready")
    expect(fetchLogs).toHaveBeenCalledTimes(Math.ceil(windowSize / 500))
    expect(fetchLogs.mock.calls.every(([params]) => params.limit === 500)).toBe(true)
    expect(result.current.lines).toHaveLength(windowSize)
    expect(result.current.lines[0]?.id).toBe("l1")
    expect(result.current.lines.at(-1)?.id).toBe(`l${windowSize}`)
    expect(result.current.hasOlder).toBe(false)
    unmount()
  })

  it("progressively prepends only unique older lines while hydrating", async () => {
    const older = deferred<ReturnType<typeof response>>()
    const fetchLogs = vi
      .fn()
      .mockResolvedValueOnce(response(["l3", "l4"], { previousCursor: "older-1", hasOlder: true }))
      .mockImplementationOnce(() => older.promise)

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, windowSize: 4, pollingEnabled: false }),
    )
    await waitForCondition(() => fetchLogs.mock.calls.length === 2)
    expect(result.current.phase).toBe("loadingOlder")
    expect(result.current.lines.map((item) => item.id)).toEqual(["l3", "l4"])

    await act(async () => {
      older.resolve(response(["l1", "l2", "l3"], { previousCursor: "", hasOlder: false }))
      await flushMicrotasks()
    })
    await waitForCondition(() => result.current.phase === "ready")
    expect(result.current.lines.map((item) => item.id)).toEqual(["l1", "l2", "l3", "l4"])
    unmount()
  })

  it("retains partial history when the older cursor is exhausted", async () => {
    const fetchLogs = vi
      .fn()
      .mockResolvedValueOnce(response(["l3", "l4"], { previousCursor: "older-1", hasOlder: true }))
      .mockResolvedValueOnce(response(["l1", "l2"], { previousCursor: "", hasOlder: false }))

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, windowSize: 5, pollingEnabled: false }),
    )

    await waitForCondition(() => result.current.phase === "ready")
    expect(fetchLogs).toHaveBeenCalledTimes(2)
    expect(result.current.lines.map((item) => item.id)).toEqual(["l1", "l2", "l3", "l4"])
    expect(result.current.hasOlder).toBe(false)
    unmount()
  })

  it("stops history hydration at a reported continuity gap", async () => {
    const fetchLogs = vi
      .fn()
      .mockResolvedValueOnce(response(["l3", "l4"], { previousCursor: "older-1", hasOlder: true }))
      .mockResolvedValueOnce(response(["l2"], {
        previousCursor: "older-2",
        hasOlder: true,
        gap: true,
        gapReason: "query_limit",
      }))

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, windowSize: 4, pollingEnabled: false }),
    )

    await waitForCondition(() => result.current.phase === "ready")
    expect(fetchLogs).toHaveBeenCalledTimes(2)
    expect(result.current.lines.map((item) => item.id)).toEqual(["l2", "l3", "l4"])
    expect(result.current.gap).toBe(true)
    expect(result.current.gapReason).toBe("query_limit")
    unmount()
  })

  it("starts fresh history hydration after a previous generation reported a gap", async () => {
    const fetchLogs = vi
      .fn()
      .mockResolvedValueOnce(response(["old-latest"], { previousCursor: "old-older", hasOlder: true }))
      .mockResolvedValueOnce(response([], {
        previousCursor: "old-older",
        hasOlder: true,
        gap: true,
        gapReason: "query_limit",
      }))
      .mockResolvedValueOnce(response(["fresh-latest"], {
        previousCursor: "fresh-older",
        hasOlder: true,
        gap: false,
      }))
      .mockResolvedValueOnce(response(["fresh-older"], { previousCursor: "", hasOlder: false, gap: false }))

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, windowSize: 2, pollingEnabled: false }),
    )
    await waitForCondition(() => result.current.phase === "ready" && result.current.gap)

    act(() => result.current.refresh())
    await waitForCondition(() => result.current.phase === "ready" && fetchLogs.mock.calls.length === 4)
    expect(fetchLogs).toHaveBeenNthCalledWith(4, expect.objectContaining({ cursor: "fresh-older", direction: "older" }))
    expect(result.current.lines.map((item) => item.id)).toEqual(["fresh-older", "fresh-latest"])
    expect(result.current.gap).toBe(false)
    unmount()
  })

  it("does not retry a non-recoverable older error and preserves the loaded lines", async () => {
    const invalidCursor = Object.assign(new Error("invalid cursor"), { code: "invalid_cursor", status: 400 })
    const fetchLogs = vi
      .fn()
      .mockResolvedValueOnce(response(["l3", "l4"], { previousCursor: "older-1", hasOlder: true }))
      .mockRejectedValueOnce(invalidCursor)

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, windowSize: 4, pollingEnabled: false }),
    )

    await waitForCondition(() => result.current.errorCode === "invalid_cursor")
    expect(fetchLogs).toHaveBeenCalledTimes(2)
    expect(result.current.lines.map((item) => item.id)).toEqual(["l3", "l4"])
    expect(result.current.errorMessage).toBe("invalid cursor")
    unmount()
  })

  it("retries recoverable history with bounded exponential backoff before retaining partial lines", async () => {
    vi.useFakeTimers()
    const fetchLogs = vi
      .fn()
      .mockResolvedValueOnce(response(["l2"], { previousCursor: "older-1", hasOlder: true }))
      .mockRejectedValue(new Error("temporary network failure"))

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, windowSize: 2, pollingEnabled: false }),
    )
    await waitForCondition(() => fetchLogs.mock.calls.length === 2)

    for (const delayMs of [250, 500, 1000]) {
      await act(async () => {
        await vi.advanceTimersByTimeAsync(delayMs)
      })
    }
    await waitForCondition(() => result.current.errorMessage === "temporary network failure")
    expect(fetchLogs).toHaveBeenCalledTimes(5)
    expect(fetchLogs.mock.calls.slice(1).every(([params]) => params.cursor === "older-1")).toBe(true)
    expect(result.current.lines.map((item) => item.id)).toEqual(["l2"])

    await act(async () => {
      await vi.advanceTimersByTimeAsync(10_000)
    })
    expect(fetchLogs).toHaveBeenCalledTimes(5)
    unmount()
  })

  it("keeps a terminal history error visible when auto refresh is disabled", async () => {
    vi.useFakeTimers()
    const fetchLogs = vi
      .fn()
      .mockResolvedValueOnce(response(["l2"], { previousCursor: "older-1", hasOlder: true }))
      .mockRejectedValue(new Error("temporary history failure"))

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, windowSize: 2, pollingEnabled: false }),
    )
    await waitForCondition(() => fetchLogs.mock.calls.length === 2)

    for (const delayMs of [250, 500, 1000]) {
      await act(async () => {
        await vi.advanceTimersByTimeAsync(delayMs)
      })
    }

    await waitForCondition(() => result.current.errorMessage === "temporary history failure")
    expect(result.current.phase).toBe("error")
    expect(result.current.lines.map((item) => item.id)).toEqual(["l2"])
    unmount()
  })

  it("keeps terminal history error detail while follow resumes", async () => {
    vi.useFakeTimers()
    const fetchLogs = vi.fn((params: FetchLogStreamParams) => {
      if (params.direction === "older") {
        return Promise.reject(new Error("temporary history failure"))
      }
      if (params.direction === "newer") {
        return Promise.resolve(response(["l3"], {
          nextCursor: "follow-2",
          hasNewer: false,
          caughtUp: true,
        }))
      }
      return Promise.resolve(response(["l2"], {
        nextCursor: "follow-1",
        previousCursor: "older-1",
        hasOlder: true,
      }))
    })

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, windowSize: 2, pollIntervalMs: 10 }),
    )
    await waitForCondition(() => fetchLogs.mock.calls.length === 2)

    for (const delayMs of [250, 500, 1000]) {
      await act(async () => {
        await vi.advanceTimersByTimeAsync(delayMs)
      })
    }

    await waitForCondition(() => result.current.phase === "following")
    expect(fetchLogs.mock.calls.some(([params]) => params.direction === "newer")).toBe(true)
    expect(result.current.errorMessage).toBe("temporary history failure")
    expect(result.current.lines.map((item) => item.id)).toEqual(["l2", "l3"])
    unmount()
  })

  it.each(["close", "grow", "manual refresh"] as const)("cancels a scheduled history retry on %s", async (action) => {
    vi.useFakeTimers()
    let latestLoads = 0
    const fetchLogs = vi.fn((params: FetchLogStreamParams) => {
      if (params.direction === "older") {
        return Promise.reject(new Error("temporary history failure"))
      }
      latestLoads += 1
      return Promise.resolve(response(
        latestLoads === 1 ? ["l2"] : ["fresh"],
        latestLoads === 1
          ? { previousCursor: "older-1", hasOlder: true }
          : { previousCursor: "", hasOlder: false },
      ))
    })

    const { result, rerender, unmount } = renderHookWithProviders(
      ({ enabled, windowSize }) => useLogStreamViewer({ enabled, fetchLogs, windowSize, pollingEnabled: false }),
      { initialProps: { enabled: true, windowSize: 2 } },
    )
    await waitForCondition(() => fetchLogs.mock.calls.length === 2 && vi.getTimerCount() === 1)

    await act(async () => {
      if (action === "close") rerender({ enabled: false, windowSize: 2 })
      if (action === "grow") rerender({ enabled: true, windowSize: 3 })
      if (action === "manual refresh") result.current.refresh()
      await flushMicrotasks()
    })
    expect(vi.getTimerCount()).toBe(0)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(10_000)
    })
    expect(fetchLogs.mock.calls.filter(([params]) => params.direction === "older")).toHaveLength(1)
    unmount()
  })

  it("drains follow backlog from the original cursor before scheduling the next poll", async () => {
    vi.useFakeTimers()
    const fetchLogs = vi
      .fn()
      .mockResolvedValueOnce(response(["l2"], { nextCursor: "follow-1", previousCursor: "older-1", hasOlder: true }))
      .mockResolvedValueOnce(response(["l1"], { nextCursor: "older-must-not-replace-follow", previousCursor: "", hasOlder: false }))
      .mockResolvedValueOnce(response(["l3"], { nextCursor: "follow-2", hasNewer: true, caughtUp: false }))
      .mockResolvedValueOnce(response(["l4"], { nextCursor: "follow-3", hasNewer: false, caughtUp: true }))
      .mockResolvedValueOnce(response([], { nextCursor: "follow-3", hasNewer: false, caughtUp: true }))

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, windowSize: 2, pollIntervalMs: 50 }),
    )

    await waitForCondition(() => result.current.phase === "following" && fetchLogs.mock.calls.length === 4)
    expect(fetchLogs).toHaveBeenNthCalledWith(3, expect.objectContaining({ cursor: "follow-1", direction: "newer", limit: 500 }))
    expect(fetchLogs).toHaveBeenNthCalledWith(4, expect.objectContaining({ cursor: "follow-2", direction: "newer", limit: 500 }))
    expect(result.current.lines.map((item) => item.id)).toEqual(["l3", "l4"])

    await act(async () => {
      await vi.advanceTimersByTimeAsync(49)
    })
    expect(fetchLogs).toHaveBeenCalledTimes(4)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1)
    })
    await waitForCondition(() => fetchLogs.mock.calls.length === 5)
    expect(fetchLogs).toHaveBeenNthCalledWith(5, expect.objectContaining({ cursor: "follow-3", direction: "newer", limit: 500 }))
    unmount()
  })

  it("preserves the loaded lines reference when a follow poll adds no unique logs", async () => {
    vi.useFakeTimers()
    const fetchLogs = vi
      .fn()
      .mockResolvedValueOnce(response(["l1"], {
        nextCursor: "follow-1",
        hasNewer: false,
        caughtUp: false,
      }))
      .mockResolvedValueOnce(response([], {
        nextCursor: "follow-1",
        hasNewer: false,
        caughtUp: true,
      }))

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, windowSize: 100, pollIntervalMs: 10 }),
    )
    await waitForCondition(() => result.current.phase === "following")
    const stableLines = result.current.lines

    await act(async () => {
      await vi.advanceTimersByTimeAsync(10)
    })
    await waitForCondition(() => fetchLogs.mock.calls.length === 2 && result.current.phase === "following")

    expect(result.current.lines).toBe(stableLines)
    expect(result.current.caughtUp).toBe(true)
    unmount()
  })

  it("rechecks latest after an empty response has no follow cursor, then resumes follow polling", async () => {
    vi.useFakeTimers()
    const fetchLogs = vi
      .fn()
      .mockResolvedValueOnce(response([], { nextCursor: "", hasNewer: false, caughtUp: true }))
      .mockResolvedValueOnce(response(["l1"], { nextCursor: "follow-1", hasNewer: false, caughtUp: true }))
      .mockResolvedValueOnce(response(["l2"], { nextCursor: "follow-2", hasNewer: false, caughtUp: true }))

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, windowSize: 2, pollIntervalMs: 10 }),
    )
    await waitForCondition(() => result.current.phase === "following")

    await act(async () => {
      await vi.advanceTimersByTimeAsync(10)
    })
    await waitForCondition(() => fetchLogs.mock.calls.length === 2)
    expect(fetchLogs).toHaveBeenNthCalledWith(2, expect.objectContaining({ limit: 500 }))
    expect(fetchLogs.mock.calls[1]?.[0].direction).toBeUndefined()

    await act(async () => {
      await vi.advanceTimersByTimeAsync(10)
    })
    await waitForCondition(() => fetchLogs.mock.calls.length === 3)
    expect(fetchLogs).toHaveBeenNthCalledWith(3, expect.objectContaining({ cursor: "follow-1", direction: "newer", limit: 500 }))
    expect(result.current.lines.map((item) => item.id)).toEqual(["l1", "l2"])
    unmount()
  })

  it("does not restart latest polling when a gap response has no follow cursor", async () => {
    vi.useFakeTimers()
    const fetchLogs = vi.fn().mockResolvedValue(response(["l1"], {
      nextCursor: "",
      hasNewer: false,
      caughtUp: false,
      gap: true,
      gapReason: "query_limit",
    }))

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, windowSize: 100, pollIntervalMs: 10 }),
    )

    await waitForCondition(() => result.current.gap && fetchLogs.mock.calls.length === 1)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(10_000)
    })

    expect(fetchLogs).toHaveBeenCalledTimes(1)
    expect(result.current.gapReason).toBe("query_limit")
    unmount()
  })

  it("keeps automatic refresh alive after a recoverable follow failure", async () => {
    vi.useFakeTimers()
    const fetchLogs = vi
      .fn()
      .mockResolvedValueOnce(response(["l1"], { nextCursor: "follow-1", hasNewer: false }))
      .mockRejectedValueOnce(new Error("temporary follow failure"))
      .mockResolvedValueOnce(response(["l2"], { nextCursor: "follow-2", hasNewer: false, caughtUp: true }))

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, windowSize: 2, pollIntervalMs: 25 }),
    )
    await waitForCondition(() => result.current.phase === "following")

    await act(async () => {
      await vi.advanceTimersByTimeAsync(25)
    })
    await waitForCondition(() => result.current.errorMessage === "temporary follow failure")
    expect(fetchLogs).toHaveBeenCalledTimes(2)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(25)
    })
    await waitForCondition(() => fetchLogs.mock.calls.length === 3)
    expect(result.current.lines.map((item) => item.id)).toEqual(["l1", "l2"])
    unmount()
  })

  it("cancels follow while disabled and immediately resumes from the retained cursor", async () => {
    vi.useFakeTimers()
    const staleFollow = deferred<ReturnType<typeof response>>()
    let staleFollowSignal: AbortSignal | undefined
    const fetchLogs = vi
      .fn()
      .mockResolvedValueOnce(response(["l1"], { nextCursor: "follow-1", hasNewer: false }))
      .mockImplementationOnce(({ signal }: { signal?: AbortSignal }) => {
        staleFollowSignal = signal
        return staleFollow.promise
      })
      .mockResolvedValueOnce(response(["l2"], { nextCursor: "follow-2", hasNewer: false, caughtUp: true }))

    const { result, rerender, unmount } = renderHookWithProviders(
      ({ pollingEnabled }) => useLogStreamViewer({
        enabled: true,
        fetchLogs,
        windowSize: 2,
        pollIntervalMs: 10,
        pollingEnabled,
      }),
      { initialProps: { pollingEnabled: true } },
    )
    await waitForCondition(() => result.current.phase === "following")

    await act(async () => {
      await vi.advanceTimersByTimeAsync(10)
    })
    await waitForCondition(() => fetchLogs.mock.calls.length === 2)

    await act(async () => {
      rerender({ pollingEnabled: false })
      await flushMicrotasks()
    })
    expect(staleFollowSignal?.aborted).toBe(true)
    await act(async () => {
      rerender({ pollingEnabled: true })
      await flushMicrotasks()
    })

    await waitForCondition(() => fetchLogs.mock.calls.length === 3)
    expect(fetchLogs).toHaveBeenNthCalledWith(3, expect.objectContaining({ cursor: "follow-1", direction: "newer" }))
    await waitForCondition(() => result.current.phase === "following")

    await act(async () => {
      staleFollow.resolve(response(["stale"], { nextCursor: "stale-cursor", hasNewer: false }))
      await flushMicrotasks()
    })
    expect(result.current.lines.map((item) => item.id)).toEqual(["l1", "l2"])
    unmount()
  })

  it("cancels in-flight history on manual refresh and rejects its late response", async () => {
    const staleHistory = deferred<ReturnType<typeof response>>()
    let staleHistorySignal: AbortSignal | undefined
    const fetchLogs = vi
      .fn()
      .mockResolvedValueOnce(response(["l2"], { previousCursor: "older-1", hasOlder: true }))
      .mockImplementationOnce(({ signal }: { signal?: AbortSignal }) => {
        staleHistorySignal = signal
        return staleHistory.promise
      })
      .mockResolvedValueOnce(response(["fresh"], { nextCursor: "follow-fresh", hasOlder: false }))

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, windowSize: 2, pollingEnabled: false }),
    )
    await waitForCondition(() => fetchLogs.mock.calls.length === 2)

    act(() => result.current.refresh())
    expect(staleHistorySignal?.aborted).toBe(true)
    await waitForCondition(() => result.current.phase === "ready")
    expect(result.current.lines.map((item) => item.id)).toEqual(["fresh"])

    await act(async () => {
      staleHistory.resolve(response(["stale"], { previousCursor: "older-2", hasOlder: true }))
      await flushMicrotasks()
    })
    expect(result.current.lines.map((item) => item.id)).toEqual(["fresh"])
    unmount()
  })

  it("keeps visible lines during a grow rebuild and rejects the previous generation", async () => {
    const staleHistory = deferred<ReturnType<typeof response>>()
    const rebuiltLatest = deferred<ReturnType<typeof response>>()
    let staleHistorySignal: AbortSignal | undefined
    const fetchLogs = vi
      .fn()
      .mockResolvedValueOnce(response(["l2", "l3"], { previousCursor: "older-1", hasOlder: true }))
      .mockImplementationOnce(({ signal }: { signal?: AbortSignal }) => {
        staleHistorySignal = signal
        return staleHistory.promise
      })
      .mockImplementationOnce(() => rebuiltLatest.promise)

    const { result, rerender, unmount } = renderHookWithProviders(
      ({ windowSize }) => useLogStreamViewer({ enabled: true, fetchLogs, windowSize, pollingEnabled: false }),
      { initialProps: { windowSize: 3 } },
    )
    await waitForCondition(() => fetchLogs.mock.calls.length === 2)

    act(() => rerender({ windowSize: 4 }))
    await waitForCondition(() => fetchLogs.mock.calls.length === 3)
    expect(staleHistorySignal?.aborted).toBe(true)
    expect(result.current.phase).toBe("initialLoading")
    expect(result.current.lines.map((item) => item.id)).toEqual(["l2", "l3"])

    await act(async () => {
      staleHistory.resolve(response(["stale"], { previousCursor: "older-2", hasOlder: true }))
      await flushMicrotasks()
    })
    expect(result.current.lines.map((item) => item.id)).toEqual(["l2", "l3"])

    await act(async () => {
      rebuiltLatest.resolve(response(["fresh-1", "fresh-2"], { hasOlder: false }))
      await flushMicrotasks()
    })
    await waitForCondition(() => result.current.phase === "ready")
    expect(result.current.lines.map((item) => item.id)).toEqual(["fresh-1", "fresh-2"])
    unmount()
  })

  it("shrinks locally to the newest lines without rebuilding the transport window", async () => {
    const fetchLogs = vi.fn().mockResolvedValue(response(["l1", "l2", "l3"], {
      previousCursor: "older-1",
      hasOlder: true,
    }))

    const { result, rerender, unmount } = renderHookWithProviders(
      ({ windowSize }) => useLogStreamViewer({ enabled: true, fetchLogs, windowSize, pollingEnabled: false }),
      { initialProps: { windowSize: 3 } },
    )
    await waitForCondition(() => result.current.phase === "ready")
    expect(fetchLogs).toHaveBeenCalledTimes(1)

    await act(async () => {
      rerender({ windowSize: 2 })
      await flushMicrotasks()
    })
    expect(result.current.lines.map((item) => item.id)).toEqual(["l2", "l3"])
    expect(result.current.hasOlder).toBe(false)
    expect(fetchLogs).toHaveBeenCalledTimes(1)
    unmount()
  })

  it("cancels an in-flight follow and immediately restarts from its retained cursor when shrinking", async () => {
    vi.useFakeTimers()
    const staleFollow = deferred<ReturnType<typeof response>>()
    let staleFollowSignal: AbortSignal | undefined
    const fetchLogs = vi
      .fn()
      .mockResolvedValueOnce(response(["l1", "l2", "l3"], { nextCursor: "follow-1", hasNewer: false }))
      .mockImplementationOnce(({ signal }: { signal?: AbortSignal }) => {
        staleFollowSignal = signal
        return staleFollow.promise
      })
      .mockResolvedValueOnce(response(["l4"], { nextCursor: "follow-2", hasNewer: false, caughtUp: true }))

    const { result, rerender, unmount } = renderHookWithProviders(
      ({ windowSize }) => useLogStreamViewer({ enabled: true, fetchLogs, windowSize, pollIntervalMs: 10 }),
      { initialProps: { windowSize: 3 } },
    )
    await waitForCondition(() => result.current.phase === "following")

    await act(async () => {
      await vi.advanceTimersByTimeAsync(10)
    })
    await waitForCondition(() => fetchLogs.mock.calls.length === 2)

    await act(async () => {
      rerender({ windowSize: 2 })
      await flushMicrotasks()
    })
    expect(staleFollowSignal?.aborted).toBe(true)

    await waitForCondition(() => fetchLogs.mock.calls.length === 3)
    expect(fetchLogs).toHaveBeenNthCalledWith(3, expect.objectContaining({
      cursor: "follow-1",
      direction: "newer",
      limit: 500,
    }))
    await waitForCondition(() => result.current.phase === "following")
    expect(result.current.lines.map((item) => item.id)).toEqual(["l3", "l4"])

    await act(async () => {
      staleFollow.resolve(response(["stale"], { nextCursor: "stale-cursor", hasNewer: false }))
      await flushMicrotasks()
    })
    expect(result.current.lines.map((item) => item.id)).toEqual(["l3", "l4"])
    unmount()
  })

  it("does not retry a wrapped network error after its history request was canceled", async () => {
    vi.useFakeTimers()
    const fetchLogs = vi.fn((params: FetchLogStreamParams) => {
      if (params.direction !== "older") {
        return Promise.resolve(response(["l2"], { previousCursor: "older-1", hasOlder: true }))
      }
      return new Promise<ReturnType<typeof response>>((_resolve, reject) => {
        params.signal?.addEventListener("abort", () => {
          reject(Object.assign(new Error("canceled by transport"), { code: "network_error" }))
        }, { once: true })
      })
    })

    const { rerender, unmount } = renderHookWithProviders(
      ({ enabled }) => useLogStreamViewer({ enabled, fetchLogs, windowSize: 2, pollingEnabled: false }),
      { initialProps: { enabled: true } },
    )
    await waitForCondition(() => fetchLogs.mock.calls.length === 2)

    act(() => rerender({ enabled: false }))
    await act(async () => {
      await flushMicrotasks()
    })
    expect(vi.getTimerCount()).toBe(0)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(10_000)
    })
    expect(fetchLogs).toHaveBeenCalledTimes(2)
    unmount()
  })

  it("preserves a disabled auto-refresh setting across manual refresh", async () => {
    vi.useFakeTimers()
    const fetchLogs = vi
      .fn()
      .mockResolvedValueOnce(response(["l1"], { nextCursor: "follow-1", hasNewer: false }))
      .mockResolvedValueOnce(response(["l2"], { nextCursor: "follow-2", hasNewer: false }))

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, windowSize: 2, pollIntervalMs: 10, pollingEnabled: false }),
    )
    await waitForCondition(() => result.current.phase === "ready")

    act(() => result.current.refresh())
    await waitForCondition(() => fetchLogs.mock.calls.length === 2 && result.current.phase === "ready")
    await act(async () => {
      await vi.advanceTimersByTimeAsync(100)
    })
    expect(fetchLogs).toHaveBeenCalledTimes(2)
    expect(result.current.lines.map((item) => item.id)).toEqual(["l2"])
    unmount()
  })

  it("keeps the latest N lines while the viewport is away from the bottom", async () => {
    vi.useFakeTimers()
    const fetchLogs = vi
      .fn()
      .mockResolvedValueOnce(response(["l1", "l2"], { nextCursor: "follow-1" }))
      .mockResolvedValueOnce(response(["l2", "l3"], { nextCursor: "follow-2" }))

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, windowSize: 2, pollIntervalMs: 10 }),
    )
    await waitForCondition(() => result.current.phase === "following")
    Object.defineProperty(result.current.viewportRef, "current", {
      configurable: true,
      value: { scrollHeight: 400, scrollTop: 0, clientHeight: 100 },
    })
    act(() => result.current.handleViewportScroll())
    expect(result.current.autoScroll).toBe(false)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(10)
    })
    await waitForCondition(() => result.current.lines.map((item) => item.id).join(",") === "l2,l3")
    expect(result.current.trimmedBefore).toBe(true)
    expect(result.current.phase).toBe("following")
    unmount()
  })

  it("does not treat virtual row measurement during automatic scrolling as a user pause", async () => {
    vi.useFakeTimers()
    const fetchLogs = vi.fn().mockResolvedValue(response(["l1", "l2"]))
    const viewport = { scrollHeight: 400, scrollTop: 0, clientHeight: 100 }

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, windowSize: 2, pollingEnabled: false }),
    )
    Object.defineProperty(result.current.viewportRef, "current", {
      configurable: true,
      value: viewport,
    })
    await waitForCondition(() => result.current.phase === "ready")

    viewport.scrollHeight = 800
    act(() => result.current.handleViewportScroll())
    expect(result.current.autoScroll).toBe(true)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(150)
    })
    expect(viewport.scrollTop).toBe(800)

    viewport.scrollTop = 0
    act(() => result.current.handleViewportScroll())
    expect(result.current.autoScroll).toBe(false)
    unmount()
  })

  it("retries a recoverable older request with the same cursor before committing it", async () => {
    vi.useFakeTimers()
    const fetchLogs = vi
      .fn()
      .mockResolvedValueOnce(response(["l2"], { previousCursor: "older-1", hasOlder: true }))
      .mockRejectedValueOnce(new Error("temporary network failure"))
      .mockResolvedValueOnce(response(["l1"], { previousCursor: "", hasOlder: false }))

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, windowSize: 2, pollingEnabled: false }),
    )
    await waitForCondition(() => fetchLogs.mock.calls.length === 2)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(250)
    })
    await waitForCondition(() => result.current.phase === "ready")
    expect(fetchLogs).toHaveBeenNthCalledWith(2, expect.objectContaining({ cursor: "older-1", direction: "older" }))
    expect(fetchLogs).toHaveBeenNthCalledWith(3, expect.objectContaining({ cursor: "older-1", direction: "older" }))
    expect(result.current.lines.map((item) => item.id)).toEqual(["l1", "l2"])
    unmount()
  })

  it("stops an in-flight history loop when the window shrinks", async () => {
    let resolveOlder: ((value: ReturnType<typeof response>) => void) | undefined
    const fetchLogs = vi
      .fn()
      .mockResolvedValueOnce(response(["l3"], { previousCursor: "older-1", hasOlder: true }))
      .mockImplementationOnce(
        () => new Promise<ReturnType<typeof response>>((resolve) => {
          resolveOlder = resolve
        }),
      )

    const { result, rerender, unmount } = renderHookWithProviders(
      ({ windowSize }) => useLogStreamViewer({ enabled: true, fetchLogs, windowSize, pollingEnabled: false }),
      { initialProps: { windowSize: 3 } },
    )
    await waitForCondition(() => fetchLogs.mock.calls.length === 2)
    await act(async () => {
      rerender({ windowSize: 1 })
      await flushMicrotasks()
    })
    await act(async () => {
      resolveOlder?.(response(["l2"], { previousCursor: "older-2", hasOlder: true }))
      await flushMicrotasks()
    })
    await waitForCondition(() => result.current.phase === "ready")
    expect(result.current.lines.map((item) => item.id)).toEqual(["l3"])
    expect(fetchLogs).toHaveBeenCalledTimes(2)
    unmount()
  })

  it("cancels and rejects an in-flight older response when the window shrinks", async () => {
    const older = deferred<ReturnType<typeof response>>()
    let olderSignal: AbortSignal | undefined
    const fetchLogs = vi
      .fn()
      .mockResolvedValueOnce(response(["l3"], { previousCursor: "older-1", hasOlder: true }))
      .mockImplementationOnce(({ signal }: { signal?: AbortSignal }) => {
        olderSignal = signal
        return older.promise
      })

    const { result, rerender, unmount } = renderHookWithProviders(
      ({ windowSize }) => useLogStreamViewer({ enabled: true, fetchLogs, windowSize, pollingEnabled: false }),
      { initialProps: { windowSize: 3 } },
    )
    await waitForCondition(() => fetchLogs.mock.calls.length === 2)

    await act(async () => {
      rerender({ windowSize: 1 })
      await flushMicrotasks()
    })
    expect(olderSignal?.aborted).toBe(true)

    await act(async () => {
      older.resolve(response(["l2"], { previousCursor: "older-2", hasOlder: true }))
      await flushMicrotasks()
    })
    expect(result.current.lines.map((item) => item.id)).toEqual(["l3"])
    expect(result.current.hasOlder).toBe(false)
    unmount()
  })

  it("ignores canceled initial requests instead of surfacing a transient error", async () => {
    const canceledError = new Error("canceled") as Error & { code: string }
    canceledError.name = "CanceledError"
    canceledError.code = "ERR_CANCELED"
    const fetchLogs = vi.fn().mockRejectedValue(canceledError)

    const { result, unmount } = renderHookWithProviders(() => useLogStreamViewer({ enabled: true, fetchLogs }))
    await act(async () => {
      await flushMicrotasks()
    })
    expect(result.current.phase).toBe("initialLoading")
    expect(result.current.errorCode).toBeNull()
    unmount()
  })

  it("uses a 100-line window by default while preserving the 500-row transport batch", async () => {
    const ids = Array.from({ length: 101 }, (_, index) => `l${index + 1}`)
    const fetchLogs = vi.fn().mockResolvedValue(response(ids, { nextCursor: "" }))

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, pollingEnabled: false }),
    )

    await waitForCondition(() => result.current.phase === "ready")
    expect(result.current.lines).toHaveLength(100)
    expect(result.current.lines[0]?.id).toBe("l2")
    expect(result.current.lines.at(-1)?.id).toBe("l101")
    expect(fetchLogs).toHaveBeenCalledWith(expect.objectContaining({ limit: 500 }))
    unmount()
  })

  it("marks an oversized initial latest page as trimmed", async () => {
    const fetchLogs = vi.fn().mockResolvedValue(response(logIds(1, 500), {
      previousCursor: "older-before-window",
      hasOlder: true,
    }))

    const { result, unmount } = renderHookWithProviders(() =>
      useLogStreamViewer({ enabled: true, fetchLogs, windowSize: 100, pollingEnabled: false }),
    )

    await waitForCondition(() => result.current.phase === "ready")
    expect(result.current.lines).toHaveLength(100)
    expect(result.current.lines[0]?.id).toBe("l401")
    expect(result.current.trimmedBefore).toBe(true)
    unmount()
  })

  it("keeps the initial request eligible when Strict Mode replays its effect", async () => {
    const fetchLogs = vi.fn(({ signal }: { signal?: AbortSignal }) => new Promise<ReturnType<typeof response>>((resolve, reject) => {
      const abort = () => reject(new DOMException("canceled", "AbortError"))
      signal?.addEventListener("abort", abort, { once: true })
      queueMicrotask(() => {
        if (!signal?.aborted) {
          resolve(response(["l1"], { nextCursor: "" }))
        }
      })
    }))

    const { result, unmount } = renderHook(
      () => useLogStreamViewer({ enabled: true, fetchLogs, pollingEnabled: false }),
      {
        wrapper: ({ children }) => React.createElement(React.StrictMode, null, children),
      },
    )

    await waitForCondition(() => result.current.phase === "ready")
    expect(fetchLogs).toHaveBeenCalled()
    expect(result.current.lines.map((item) => item.id)).toEqual(["l1"])
    unmount()
  })

  it("resets initialization when canceling the unmounted generation", () => {
    expect(source).toContain("React development Strict Mode replays effects")
    expect(source).toContain("initializedRef.current = false")
    expect(source).toContain("cancelGeneration()")
  })
})
