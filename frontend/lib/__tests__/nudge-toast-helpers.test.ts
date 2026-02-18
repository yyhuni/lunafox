import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import {
  isLocalStorageAvailable,
  isNudgeSuppressed,
  suppressNudge,
  withNudgeDismiss,
} from "@/lib/nudge-toast-helpers"

describe("nudge toast helpers", () => {
  beforeEach(() => {
    localStorage.clear()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it("detects localStorage availability in browser environments", () => {
    expect(isLocalStorageAvailable()).toBe(true)
  })

  it("handles permanent suppression", () => {
    expect(isNudgeSuppressed("nudge-key")).toBe(false)
    suppressNudge("nudge-key")
    expect(isNudgeSuppressed("nudge-key")).toBe(true)
  })

  it("handles cooldown suppression and expiry", () => {
    vi.spyOn(Date, "now").mockReturnValue(1_000)
    suppressNudge("cooldown-key", 2_000)
    expect(isNudgeSuppressed("cooldown-key", 2_000)).toBe(true)

    vi.spyOn(Date, "now").mockReturnValue(4_000)
    expect(isNudgeSuppressed("cooldown-key", 2_000)).toBe(false)
    expect(localStorage.getItem("cooldown-key")).toBeNull()
  })

  it("wraps actions to dismiss after click", () => {
    const onClick = vi.fn()
    const onDismiss = vi.fn()
    const wrapped = withNudgeDismiss({ label: "Go", onClick }, onDismiss)

    wrapped?.onClick?.()

    expect(onClick).toHaveBeenCalledTimes(1)
    expect(onDismiss).toHaveBeenCalledTimes(1)
  })
})
