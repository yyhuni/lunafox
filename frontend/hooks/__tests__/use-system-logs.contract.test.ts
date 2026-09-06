import { act, cleanup } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

import { useSystemLogs } from "@/hooks/use-system-logs"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-system-logs.ts"), "utf8")

const serviceMocks = vi.hoisted(() => ({
  fetchServerLogs: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

vi.mock("@/services/system-log.service", () => ({
  systemLogService: serviceMocks,
  SystemLogQueryError: class SystemLogQueryError extends Error {
    code: string
    status?: number

    constructor(code: string, message: string, status?: number) {
      super(message)
      this.name = "SystemLogQueryError"
      this.code = code
      this.status = status
    }
  },
}))

async function flushMicrotasks() {
  await Promise.resolve()
  await Promise.resolve()
}

async function waitForCondition(predicate: () => boolean, attempts = 12) {
  for (let index = 0; index < attempts; index += 1) {
    if (predicate()) {
      return
    }
    await act(async () => {
      await flushMicrotasks()
    })
  }
  expect(predicate()).toBe(true)
}

describe("use-system-logs contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
    vi.clearAllTimers()
    vi.useRealTimers()
  })

  it("preserves current source markers", () => {
    expect(source).toContain("export function useSystemLogs")
    expect(source).toContain("fetchServerLogs")
    expect(source).toContain("const DEFAULT_LOG_WINDOW_SIZE = 100")
    expect(source).toContain("from \"react\"")
  })

  it("keeps loaded logs stable when auto refresh is toggled", async () => {
    vi.useFakeTimers()

    serviceMocks.fetchServerLogs
      .mockResolvedValueOnce({
        logs: [
          {
            id: "l1",
            ts: "2026-06-19T08:00:01Z",
            stream: "stderr",
            line: "initial log",
            truncated: false,
          },
        ],
        nextCursor: "follow-1",
        previousCursor: "",
        hasOlder: false,
        hasNewer: false,
        caughtUp: true,
        gap: false,
        gapReason: "",
      })
      .mockResolvedValueOnce({
        logs: [
          {
            id: "l2",
            ts: "2026-06-19T08:00:02Z",
            stream: "stderr",
            line: "newer log",
            truncated: false,
          },
        ],
        nextCursor: "follow-2",
        previousCursor: "",
        hasOlder: false,
        hasNewer: false,
        caughtUp: true,
        gap: false,
        gapReason: "",
      })

    const { rerender, result, unmount } = renderHookWithProviders(
      ({ autoRefresh }) =>
        useSystemLogs({
          autoRefresh,
          windowSize: 500,
        }),
      {
        initialProps: {
          autoRefresh: true,
        },
      },
    )

    await waitForCondition(() => result.current.lines.length === 1)

    rerender({ autoRefresh: false })
    await act(async () => {
      vi.advanceTimersByTime(2100)
      await flushMicrotasks()
    })

    expect(result.current.lines.map((item) => item.id)).toEqual(["l1"])
    expect(result.current.isLoading).toBe(false)
    expect(serviceMocks.fetchServerLogs).toHaveBeenCalledTimes(1)

    rerender({ autoRefresh: true })
    await act(async () => {
      vi.advanceTimersByTime(2000)
      await flushMicrotasks()
    })

    await waitForCondition(() => result.current.lines.map((item) => item.id).includes("l2"))
    expect(result.current.lines.map((item) => item.id)).toEqual(["l1", "l2"])
    expect(serviceMocks.fetchServerLogs).toHaveBeenCalledTimes(2)
    unmount()
  })

  it("does not show fetch failure or recovery toasts for canceled system log requests", async () => {
    const canceledError = new Error("canceled") as Error & { code: string }
    canceledError.name = "CanceledError"
    canceledError.code = "ERR_CANCELED"
    serviceMocks.fetchServerLogs.mockRejectedValue(canceledError)

    const { result, unmount } = renderHookWithProviders(() =>
      useSystemLogs({
        autoRefresh: true,
        windowSize: 500,
      }),
    )

    await act(async () => {
      await flushMicrotasks()
    })

    expect(result.current.isError).toBe(false)
    expect(toastMocks.error).not.toHaveBeenCalledWith("toast.systemLog.fetch.error")
    expect(toastMocks.success).not.toHaveBeenCalledWith("toast.systemLog.fetch.recovered")
    unmount()
  })
})
