import { renderHook } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const authServiceMocks = vi.hoisted(() => ({
  ensureFreshAuthToken: vi.fn(),
  authTokenRefreshDelayMs: vi.fn(),
}))

vi.mock("@/services/auth.service", () => ({
  ensureFreshAuthToken: authServiceMocks.ensureFreshAuthToken,
  authTokenRefreshDelayMs: authServiceMocks.authTokenRefreshDelayMs,
}))

import { sessionRenewalDelayMs, useSessionRenewal } from "@/hooks/use-session-renewal"

describe("useSessionRenewal", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date("2026-08-18T12:00:00.000Z"))
    vi.clearAllMocks()
    Object.defineProperty(document, "visibilityState", {
      configurable: true,
      value: "visible",
    })
    authServiceMocks.ensureFreshAuthToken.mockResolvedValue("current-token")
    authServiceMocks.authTokenRefreshDelayMs.mockReturnValue(60_000)
  })

  afterEach(() => {
    Object.defineProperty(document, "visibilityState", {
      configurable: true,
      value: "visible",
    })
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it("在提前窗口调度检查，并从每次检查返回的 token 重新计算下一次", async () => {
    authServiceMocks.ensureFreshAuthToken
      .mockResolvedValueOnce("initial-token")
      .mockResolvedValueOnce("renewed-token")
    authServiceMocks.authTokenRefreshDelayMs
      .mockReturnValueOnce(60_000)
      .mockReturnValueOnce(120_000)

    const { unmount } = renderHook(() => useSessionRenewal())

    await vi.advanceTimersByTimeAsync(0)
    expect(authServiceMocks.ensureFreshAuthToken).toHaveBeenCalledTimes(1)
    expect(authServiceMocks.authTokenRefreshDelayMs).toHaveBeenLastCalledWith(
      "initial-token",
      expect.any(Number)
    )

    await vi.advanceTimersByTimeAsync(60_000)
    expect(authServiceMocks.ensureFreshAuthToken).toHaveBeenCalledTimes(2)
    expect(authServiceMocks.authTokenRefreshDelayMs).toHaveBeenLastCalledWith(
      "renewed-token",
      expect.any(Number)
    )

    await vi.advanceTimersByTimeAsync(119_999)
    expect(authServiceMocks.ensureFreshAuthToken).toHaveBeenCalledTimes(2)

    unmount()
  })

  it("在页面重新可见或网络恢复时立即重新检查", async () => {
    const { unmount } = renderHook(() => useSessionRenewal())

    await vi.advanceTimersByTimeAsync(0)
    document.dispatchEvent(new Event("visibilitychange"))
    window.dispatchEvent(new Event("online"))
    await vi.advanceTimersByTimeAsync(0)

    expect(authServiceMocks.ensureFreshAuthToken).toHaveBeenCalledTimes(3)

    unmount()
  })

  it("页面隐藏时不因为 visibilitychange 触发续签", async () => {
    Object.defineProperty(document, "visibilityState", {
      configurable: true,
      value: "hidden",
    })
    const { unmount } = renderHook(() => useSessionRenewal())

    await vi.advanceTimersByTimeAsync(0)
    document.dispatchEvent(new Event("visibilitychange"))
    await vi.advanceTimersByTimeAsync(0)

    expect(authServiceMocks.ensureFreshAuthToken).toHaveBeenCalledTimes(1)

    unmount()
    Object.defineProperty(document, "visibilityState", {
      configurable: true,
      value: "visible",
    })
  })

  it("卸载后清理定时器和事件监听", async () => {
    const { unmount } = renderHook(() => useSessionRenewal())

    await vi.advanceTimersByTimeAsync(0)
    unmount()
    await vi.advanceTimersByTimeAsync(60_000)
    document.dispatchEvent(new Event("visibilitychange"))
    window.dispatchEvent(new Event("online"))
    await vi.advanceTimersByTimeAsync(0)

    expect(authServiceMocks.ensureFreshAuthToken).toHaveBeenCalledTimes(1)
  })

  it("令牌仍在提前窗口内时保留最小重排间隔", () => {
    authServiceMocks.authTokenRefreshDelayMs.mockReturnValue(0)

    expect(sessionRenewalDelayMs("near-expiry-token")).toBe(1_000)
  })
})
