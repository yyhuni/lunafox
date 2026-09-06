import { act, renderHook } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { useStagedEdgePanelOpen } from "@/lib/ui/use-staged-edge-panel-open"

describe("useStagedEdgePanelOpen", () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it("commits a controlled edge panel's closed state before its first opening frame", () => {
    let animationFrame: FrameRequestCallback | undefined
    const requestFrame = vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback) => {
      animationFrame = callback
      return 1
    })

    const { result, rerender } = renderHook(
      ({ open }) => useStagedEdgePanelOpen(open),
      { initialProps: { open: false } }
    )

    rerender({ open: true })

    expect(result.current).toBe(false)
    expect(requestFrame).toHaveBeenCalledTimes(1)

    act(() => {
      animationFrame?.(0)
    })

    expect(result.current).toBe(true)
  })

  it("preserves the immediate final state for users requesting reduced motion", () => {
    vi.stubGlobal("matchMedia", vi.fn().mockReturnValue({ matches: true }))
    const requestFrame = vi.spyOn(window, "requestAnimationFrame")

    const { result } = renderHook(() => useStagedEdgePanelOpen(true))

    expect(result.current).toBe(true)
    expect(requestFrame).not.toHaveBeenCalled()
  })

  it("does not change uncontrolled edge panels", () => {
    const { result } = renderHook(() => useStagedEdgePanelOpen(undefined))

    expect(result.current).toBeUndefined()
  })
})
