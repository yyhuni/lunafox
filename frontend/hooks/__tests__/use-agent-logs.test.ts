import { act, cleanup } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { useAgentLogs } from "@/hooks/use-agent-logs"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"

const serviceMocks = vi.hoisted(() => ({ fetchDistributedLogs: vi.fn() }))

vi.mock("@/services/agent.service", () => ({ agentService: serviceMocks }))

const agentNode = { id: 7 } as never
const log = (id: string) => ({ id, ts: id, tsNs: id, stream: "stdout", line: id, truncated: false })
const result = (ids: string[], overrides = {}) => ({
  logs: ids.map(log),
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

describe("useAgentLogs", () => {
  beforeEach(() => vi.clearAllMocks())
  afterEach(() => {
    cleanup()
    vi.clearAllTimers()
    vi.useRealTimers()
  })

  it("uses 500-row transport batches while hydrating the selected window", async () => {
    serviceMocks.fetchDistributedLogs
      .mockResolvedValueOnce(result(["l2"], { previousCursor: "older-1", hasOlder: true }))
      .mockResolvedValueOnce(result(["l1"], { previousCursor: "", hasOlder: false }))
      .mockResolvedValueOnce(result([], { nextCursor: "follow-1" }))

    const { result: hookResult, unmount } = renderHookWithProviders(() =>
      useAgentLogs({ open: true, agentNode, container: "lunafox-agent", windowSize: 2 }),
    )
    await waitForCondition(() => hookResult.current.phase === "following")
    expect(hookResult.current.lines.map((item) => item.id)).toEqual(["l1", "l2"])
    expect(serviceMocks.fetchDistributedLogs).toHaveBeenNthCalledWith(1, expect.objectContaining({ limit: 500 }))
    expect(serviceMocks.fetchDistributedLogs).toHaveBeenNthCalledWith(2, expect.objectContaining({ limit: 500, direction: "older" }))
    unmount()
  })

  it("does not follow after hydration while auto refresh is disabled", async () => {
    serviceMocks.fetchDistributedLogs
      .mockResolvedValueOnce(result(["l2"], { previousCursor: "older-1", hasOlder: true }))
      .mockResolvedValueOnce(result(["l1"], { previousCursor: "", hasOlder: false }))

    const { result: hookResult, unmount } = renderHookWithProviders(() =>
      useAgentLogs({ open: true, agentNode, container: "lunafox-agent", windowSize: 2, pollingEnabled: false }),
    )
    await waitForCondition(() => hookResult.current.phase === "ready")
    expect(serviceMocks.fetchDistributedLogs).toHaveBeenCalledTimes(2)
    unmount()
  })

  it("starts a fresh latest generation when the window grows", async () => {
    serviceMocks.fetchDistributedLogs
      .mockResolvedValueOnce(result(["l2"]))
      .mockResolvedValueOnce(result(["l2", "l3"], { previousCursor: "older-1", hasOlder: true }))
      .mockResolvedValueOnce(result(["l1"], { previousCursor: "", hasOlder: false }))
      .mockResolvedValueOnce(result([], { nextCursor: "follow-2" }))

    const { result: hookResult, rerender, unmount } = renderHookWithProviders(
      ({ windowSize }) => useAgentLogs({ open: true, agentNode, container: "lunafox-agent", windowSize }),
      { initialProps: { windowSize: 2 } },
    )
    await waitForCondition(() => hookResult.current.phase === "following")
    rerender({ windowSize: 3 })
    await waitForCondition(() => hookResult.current.lines.map((item) => item.id).join(",") === "l1,l2,l3")
    expect(serviceMocks.fetchDistributedLogs).toHaveBeenNthCalledWith(2, expect.objectContaining({ direction: undefined, limit: 500 }))
    unmount()
  })
})
