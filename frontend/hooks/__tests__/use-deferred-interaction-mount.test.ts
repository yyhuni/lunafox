import { act, renderHook } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { useDeferredInteractionMount } from "@/hooks/use-deferred-interaction-mount"

describe("useDeferredInteractionMount", () => {
  afterEach(() => {
    vi.clearAllTimers()
    vi.useRealTimers()
  })

  it("keeps the existing permanent mount behavior by default", () => {
    const { result, rerender } = renderHook(
      ({ active }) => useDeferredInteractionMount(active),
      { initialProps: { active: false } }
    )

    expect(result.current).toBe(false)

    rerender({ active: true })
    expect(result.current).toBe(true)

    rerender({ active: false })
    expect(result.current).toBe(true)
  })

  it("releases the mounted tree after the configured inactive delay", () => {
    vi.useFakeTimers()

    const { result, rerender } = renderHook(
      ({ active }) => useDeferredInteractionMount(active, { unmountDelayMs: 120 }),
      { initialProps: { active: true } }
    )

    expect(result.current).toBe(true)

    rerender({ active: false })
    expect(result.current).toBe(true)

    act(() => {
      vi.advanceTimersByTime(119)
    })
    expect(result.current).toBe(true)

    act(() => {
      vi.advanceTimersByTime(1)
    })
    expect(result.current).toBe(false)
  })

  it("cancels delayed release when the interaction becomes active again", () => {
    vi.useFakeTimers()

    const { result, rerender } = renderHook(
      ({ active }) => useDeferredInteractionMount(active, { unmountDelayMs: 120 }),
      { initialProps: { active: true } }
    )

    rerender({ active: false })
    act(() => {
      vi.advanceTimersByTime(60)
    })

    rerender({ active: true })
    act(() => {
      vi.advanceTimersByTime(120)
    })

    expect(result.current).toBe(true)
  })

  it("mounts and preloads when the caller provides a preload signal", () => {
    const preload = vi.fn()

    const { result, rerender } = renderHook(
      ({ preloadWhen }) =>
        useDeferredInteractionMount(false, {
          preload,
          preloadWhen,
        }),
      { initialProps: { preloadWhen: false } }
    )

    expect(result.current).toBe(false)
    expect(preload).not.toHaveBeenCalled()

    rerender({ preloadWhen: true })

    expect(result.current).toBe(true)
    expect(preload).toHaveBeenCalledTimes(1)

    rerender({ preloadWhen: true })
    expect(preload).toHaveBeenCalledTimes(1)
  })
})
