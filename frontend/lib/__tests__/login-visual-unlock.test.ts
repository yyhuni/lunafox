import { act, renderHook } from "@testing-library/react"
import { beforeEach, describe, expect, it } from "vitest"

import {
  isLoginVisualUnlocked,
  LOGIN_VISUAL_UNLOCK_STORAGE_KEY,
  unlockLoginVisual,
  useLoginVisualUnlocked,
} from "@/lib/login-visual-unlock"

describe("login visual unlock", () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it("starts hidden and persists the GitHub-triggered unlock", () => {
    expect(isLoginVisualUnlocked()).toBe(false)

    expect(unlockLoginVisual()).toBe(true)
    expect(localStorage.getItem(LOGIN_VISUAL_UNLOCK_STORAGE_KEY)).toBe("true")
    expect(isLoginVisualUnlocked()).toBe(true)
    expect(unlockLoginVisual()).toBe(false)
  })

  it("notifies the current tab when the unlock is stored", () => {
    const { result } = renderHook(() => useLoginVisualUnlocked())

    expect(result.current).toBe(false)

    act(() => {
      unlockLoginVisual()
    })

    expect(result.current).toBe(true)
  })
})
