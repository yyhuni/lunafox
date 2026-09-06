import { act, renderHook } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { usePageRefreshTimestamp } from "@/hooks/_shared/use-page-refresh-timestamp"

describe("usePageRefreshTimestamp", () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it("records the browser time after the first client mount", () => {
    const mountedAt = new Date("2026-08-03T08:09:10.000Z")
    vi.useFakeTimers()
    vi.setSystemTime(mountedAt)

    const { result } = renderHook(() => usePageRefreshTimestamp())

    expect(result.current.lastRefreshedAt).toEqual(mountedAt)
  })

  it("records a later time when a coordinated refresh completes", () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date("2026-08-03T08:09:10.000Z"))
    const { result } = renderHook(() => usePageRefreshTimestamp())
    const refreshedAt = new Date("2026-08-03T08:10:11.000Z")

    act(() => {
      vi.setSystemTime(refreshedAt)
      result.current.markRefreshCompleted()
    })

    expect(result.current.lastRefreshedAt).toEqual(refreshedAt)
  })

  it("does not write state after the page unmounts", () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date("2026-08-03T08:09:10.000Z"))
    const { result, unmount } = renderHook(() => usePageRefreshTimestamp())
    const markRefreshCompleted = result.current.markRefreshCompleted

    unmount()

    expect(() => markRefreshCompleted()).not.toThrow()
  })
})
