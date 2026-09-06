import { act, renderHook } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { useInteractionOpenLoader } from "@/components/shared/loading/use-interaction-open-loader"

describe("useInteractionOpenLoader", () => {
  it("opens after a fast dynamic loader resolves", async () => {
    const load = vi.fn().mockResolvedValue(undefined)
    const onLoaded = vi.fn()
    const { result } = renderHook(() => useInteractionOpenLoader<"create-dialog">())

    await act(async () => {
      await result.current.openAfterLoad("create-dialog", load, onLoaded)
    })

    expect(load).toHaveBeenCalledTimes(1)
    expect(onLoaded).toHaveBeenCalledTimes(1)
    expect(result.current.pendingInteraction).toBeNull()
  })

  it("does not open when the pending interaction is canceled before loading finishes", async () => {
    let finishLoad!: () => void
    const load = vi.fn(
      () => new Promise<void>((resolve) => {
        finishLoad = resolve
      })
    )
    const onLoaded = vi.fn()
    const { result } = renderHook(() => useInteractionOpenLoader<"create-dialog">())

    await act(async () => {
      void result.current.openAfterLoad("create-dialog", load, onLoaded)
    })
    expect(result.current.pendingInteraction).toBe("create-dialog")

    await act(async () => {
      result.current.cancelPending("create-dialog")
      finishLoad()
    })

    expect(load).toHaveBeenCalledTimes(1)
    expect(onLoaded).not.toHaveBeenCalled()
    expect(result.current.pendingInteraction).toBeNull()
  })
})
