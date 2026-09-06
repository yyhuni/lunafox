import { act, waitFor } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import {
  collectNewScanFailureNotifications,
  consumeNotificationRefreshStream,
  createNotificationNameBaseline,
  NOTIFICATION_STREAM_IDLE_TIMEOUT_MS,
  notificationReconnectDelayMs,
  notificationTokenExpirationMs,
  showScanFailureNotification,
  useNotificationSSE,
} from "@/hooks/use-notification-sse"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import type { NotificationInboxItem } from "@/types/notification.types"

const notificationServiceMocks = vi.hoisted(() => ({
  getStreamToken: vi.fn(),
  getFreshStreamToken: vi.fn(),
  renewStreamToken: vi.fn(),
  handleStreamSessionFailure: vi.fn(),
  list: vi.fn(),
  openStream: vi.fn(),
  streamErrorReason: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  error: vi.fn(),
}))

vi.mock("@/services/notification.service", () => ({
  NotificationService: notificationServiceMocks,
}))

vi.mock("@/lib/toast-helpers", () => ({
  toastFeedback: toastMocks,
}))

function notificationFixture(overrides: Partial<NotificationInboxItem> = {}): NotificationInboxItem {
  return {
    name: "users/current/notifications/1",
    kind: "scan-failed",
    category: "scan",
    priority: "high",
    subject: "scans/1",
    title: "Scan failed",
    message: "The scan exceeded its execution time limit.",
    occurredAt: "2026-08-27T00:00:00.000Z",
    createdAt: "2026-08-27T00:00:01.000Z",
    readAt: null,
    ...overrides,
  }
}

function tokenWithExpiration(expirationSeconds: number) {
  const payload = btoa(JSON.stringify({ exp: expirationSeconds }))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "")
  return `header.${payload}.signature`
}

function openRefreshStreamResponse() {
  return new Response(
    new ReadableStream({
      start(controller) {
        controller.enqueue(new TextEncoder().encode("event: refresh\n\n"))
      },
    }),
    {
      status: 200,
      headers: { "content-type": "text/event-stream" },
    }
  )
}

describe("use-notification-sse", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    notificationServiceMocks.list.mockResolvedValue({
      results: [],
      nextPageToken: "",
      totalSize: 0,
    })
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it("只接受 JWT 的数值 exp 并转换为毫秒", () => {
    expect(notificationTokenExpirationMs(tokenWithExpiration(1_778_000_123))).toBe(1_778_000_123_000)
    expect(notificationTokenExpirationMs("header.invalid.signature")).toBeNull()
    expect(notificationTokenExpirationMs("header.eyJleHAiOiJub3QtYS1udW1iZXIifQ.signature")).toBeNull()
  })

  it("使用有上限的 full-jitter 重连延迟", () => {
    expect(notificationReconnectDelayMs(0, () => 0)).toBe(0)
    expect(notificationReconnectDelayMs(0, () => 1)).toBe(1_000)
    expect(notificationReconnectDelayMs(5, () => 1)).toBe(30_000)
    expect(notificationReconnectDelayMs(99, () => 0.5)).toBe(15_000)
  })

  it("只为 refresh SSE 事件触发权威回读", async () => {
    const onRefresh = vi.fn().mockResolvedValue(undefined)
    const onActivity = vi.fn()
    const stream = new ReadableStream({
      start(controller) {
        controller.enqueue(new TextEncoder().encode(": heartbeat\n\n"))
        controller.enqueue(new TextEncoder().encode("event: ignored\n\n"))
        controller.enqueue(new TextEncoder().encode("event: refresh\r\n\r\n"))
        controller.close()
      },
    })

    await consumeNotificationRefreshStream(
      new Response(stream, { headers: { "content-type": "text/event-stream" } }),
      onRefresh,
      onActivity
    )

    expect(onRefresh).toHaveBeenCalledTimes(1)
    expect(onActivity).toHaveBeenCalledTimes(3)
  })

  it("初次和重连只建立基线，后续 refresh 仅提示新增的高优先级扫描失败", () => {
    const historical = notificationFixture()
    const baseline = createNotificationNameBaseline([historical])
    expect([...baseline]).toEqual([historical.name])
    expect(toastMocks.error).not.toHaveBeenCalled()

    const newFailure = notificationFixture({
      name: "users/current/notifications/2",
      subject: "scans/2",
      title: "Website discovery failed",
      message: "The task exceeded its execution time limit.",
    })
    const normalFailure = notificationFixture({
      name: "users/current/notifications/3",
      priority: "normal",
    })
    const agentOffline = notificationFixture({
      name: "users/current/notifications/4",
      kind: "agent-offline",
      category: "system",
    })
    const refreshed = collectNewScanFailureNotifications(
      baseline,
      [historical, newFailure, normalFailure, agentOffline]
    )

    expect(refreshed.newFailures).toEqual([newFailure])
    showScanFailureNotification(refreshed.newFailures[0]!)
    expect(toastMocks.error).toHaveBeenCalledWith(newFailure.title, {
      description: newFailure.message,
      id: newFailure.name,
    })

    const repeated = collectNewScanFailureNotifications(
      refreshed.observedNames,
      [historical, newFailure, normalFailure, agentOffline]
    )
    expect(repeated.newFailures).toEqual([])

    const reconnectBaseline = createNotificationNameBaseline([
      historical,
      newFailure,
      normalFailure,
      agentOffline,
    ])
    expect(collectNewScanFailureNotifications(reconnectBaseline, [historical, newFailure]).newFailures).toEqual([])
    expect(toastMocks.error).toHaveBeenCalledTimes(1)
  })

  it("收到 heartbeat 后重置 idle watchdog，超时后中止并重连回读", async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date("2026-08-26T12:00:00.000Z"))
    vi.spyOn(Math, "random").mockReturnValue(0.5)
    const token = tokenWithExpiration(Math.floor(Date.now() / 1_000) + 7_200)
    const streamControllers: ReadableStreamDefaultController<Uint8Array>[] = []
    const signals: AbortSignal[] = []
    notificationServiceMocks.getFreshStreamToken.mockResolvedValue(token)
    notificationServiceMocks.openStream.mockImplementation((_, signal: AbortSignal) => {
      signals.push(signal)
      return Promise.resolve(new Response(
        new ReadableStream<Uint8Array>({
          start(controller) {
            streamControllers.push(controller)
            signal.addEventListener("abort", () => {
              controller.error(new DOMException("Aborted", "AbortError"))
            }, { once: true })
          },
        }),
        { headers: { "content-type": "text/event-stream" } }
      ))
    })

    const { queryClient, unmount } = renderHookWithProviders(() => useNotificationSSE())
    const refetchSpy = vi.spyOn(queryClient, "refetchQueries")
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0)
    })
    expect(notificationServiceMocks.openStream).toHaveBeenCalledTimes(1)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(30_000)
      streamControllers[0]?.enqueue(new TextEncoder().encode(": heartbeat\n\n"))
      await vi.advanceTimersByTimeAsync(NOTIFICATION_STREAM_IDLE_TIMEOUT_MS - 1)
    })
    expect(signals[0]?.aborted).toBe(false)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1)
      await Promise.resolve()
    })
    expect(signals[0]?.aborted).toBe(true)
    await act(async () => {
      await vi.advanceTimersByTimeAsync(499)
    })
    expect(notificationServiceMocks.openStream).toHaveBeenCalledTimes(1)
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1)
    })
    expect(notificationServiceMocks.openStream).toHaveBeenCalledTimes(2)
    expect(refetchSpy).toHaveBeenCalledTimes(2)

    unmount()
  })

  it("离线中止连接后不启动 idle 重连", async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date("2026-08-26T12:00:00.000Z"))
    const token = tokenWithExpiration(Math.floor(Date.now() / 1_000) + 7_200)
    const signals: AbortSignal[] = []
    notificationServiceMocks.getFreshStreamToken.mockResolvedValue(token)
    notificationServiceMocks.openStream.mockImplementation((_, signal: AbortSignal) => {
      signals.push(signal)
      return Promise.resolve(new Response(
        new ReadableStream<Uint8Array>({
          start(controller) {
            signal.addEventListener("abort", () => {
              controller.error(new DOMException("Aborted", "AbortError"))
            }, { once: true })
          },
        }),
        { headers: { "content-type": "text/event-stream" } }
      ))
    })

    const { result, unmount } = renderHookWithProviders(() => useNotificationSSE())
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0)
      window.dispatchEvent(new Event("offline"))
      await vi.advanceTimersByTimeAsync(NOTIFICATION_STREAM_IDLE_TIMEOUT_MS * 2)
    })

    expect(signals[0]?.aborted).toBe(true)
    expect(notificationServiceMocks.openStream).toHaveBeenCalledTimes(1)
    expect(result.current.status).toBe("offline")
    unmount()
  })

  it("离线中止尚未释放时恢复在线，释放后立即重建流", async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date("2026-08-26T12:00:00.000Z"))
    const token = tokenWithExpiration(Math.floor(Date.now() / 1_000) + 7_200)
    notificationServiceMocks.getFreshStreamToken.mockResolvedValue(token)
    notificationServiceMocks.openStream.mockImplementation((_, signal: AbortSignal) => Promise.resolve(new Response(
      new ReadableStream<Uint8Array>({
        start(controller) {
          signal.addEventListener("abort", () => {
            controller.error(new DOMException("Aborted", "AbortError"))
          }, { once: true })
        },
      }),
      { headers: { "content-type": "text/event-stream" } }
    )))

    const { queryClient, unmount } = renderHookWithProviders(() => useNotificationSSE())
    const refetchSpy = vi.spyOn(queryClient, "refetchQueries")
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0)
      window.dispatchEvent(new Event("offline"))
      window.dispatchEvent(new Event("online"))
      await Promise.resolve()
      await vi.advanceTimersByTimeAsync(0)
    })

    expect(notificationServiceMocks.openStream).toHaveBeenCalledTimes(2)
    expect(refetchSpy).toHaveBeenCalledTimes(2)
    unmount()
  })

  it("仅在 TOKEN_EXPIRED 后续期一次，并使用新 token 重建流", async () => {
    const expiredToken = tokenWithExpiration(Math.floor(Date.now() / 1_000) + 3_600)
    const renewedToken = tokenWithExpiration(Math.floor(Date.now() / 1_000) + 7_200)
    notificationServiceMocks.getFreshStreamToken
      .mockResolvedValueOnce(expiredToken)
      .mockResolvedValueOnce(renewedToken)
    notificationServiceMocks.renewStreamToken.mockResolvedValue(renewedToken)
    notificationServiceMocks.openStream
      .mockResolvedValueOnce(new Response(null, { status: 401 }))
      .mockResolvedValueOnce(openRefreshStreamResponse())
    notificationServiceMocks.streamErrorReason.mockResolvedValue("TOKEN_EXPIRED")

    const { queryClient, unmount } = renderHookWithProviders(() => useNotificationSSE())
    const refetchSpy = vi.spyOn(queryClient, "refetchQueries")

    await waitFor(() => {
      expect(notificationServiceMocks.openStream).toHaveBeenCalledTimes(2)
    })

    expect(notificationServiceMocks.renewStreamToken).toHaveBeenCalledTimes(1)
    expect(notificationServiceMocks.openStream).toHaveBeenNthCalledWith(
      1,
      expiredToken,
      expect.any(AbortSignal)
    )
    expect(notificationServiceMocks.openStream).toHaveBeenNthCalledWith(
      2,
      renewedToken,
      expect.any(AbortSignal)
    )
    await waitFor(() => {
      expect(refetchSpy).toHaveBeenCalledWith({
        queryKey: ["notification-inbox"],
        type: "active",
      })
    })
    expect(notificationServiceMocks.handleStreamSessionFailure).not.toHaveBeenCalled()

    unmount()
  })

  it("续期失败后停止流并交给既有登录流程，而不是继续重连", async () => {
    const expiredToken = tokenWithExpiration(Math.floor(Date.now() / 1_000) + 3_600)
    notificationServiceMocks.getFreshStreamToken.mockResolvedValue(expiredToken)
    notificationServiceMocks.openStream.mockResolvedValue(new Response(null, { status: 401 }))
    notificationServiceMocks.streamErrorReason.mockResolvedValue("TOKEN_EXPIRED")
    notificationServiceMocks.renewStreamToken.mockRejectedValue(new Error("renewal failed"))

    const { result, unmount } = renderHookWithProviders(() => useNotificationSSE())

    await waitFor(() => {
      expect(notificationServiceMocks.handleStreamSessionFailure).toHaveBeenCalledTimes(1)
      expect(result.current.status).toBe("stopped")
    })
    expect(notificationServiceMocks.openStream).toHaveBeenCalledTimes(1)
    expect(notificationServiceMocks.renewStreamToken).toHaveBeenCalledTimes(1)

    unmount()
  })

  it("在打开流之前使用共享 freshness owner 返回的 token", async () => {
    const renewedToken = tokenWithExpiration(Math.floor(Date.now() / 1_000) + 7_200)
    notificationServiceMocks.getFreshStreamToken.mockResolvedValue(renewedToken)
    notificationServiceMocks.openStream.mockResolvedValue(openRefreshStreamResponse())

    const { unmount } = renderHookWithProviders(() => useNotificationSSE())

    await waitFor(() => {
      expect(notificationServiceMocks.openStream).toHaveBeenCalledWith(
        renewedToken,
        expect.any(AbortSignal)
      )
    })

    expect(notificationServiceMocks.getFreshStreamToken).toHaveBeenCalledTimes(1)
    expect(notificationServiceMocks.renewStreamToken).not.toHaveBeenCalled()

    unmount()
  })

  it("连接前 freshness 失败时停止且不进入重连循环", async () => {
    notificationServiceMocks.getFreshStreamToken.mockRejectedValue(new Error("renewal failed"))

    const { result, unmount } = renderHookWithProviders(() => useNotificationSSE())

    await waitFor(() => {
      expect(result.current.status).toBe("stopped")
    })

    expect(notificationServiceMocks.openStream).not.toHaveBeenCalled()
    expect(notificationServiceMocks.renewStreamToken).not.toHaveBeenCalled()
    expect(notificationServiceMocks.handleStreamSessionFailure).not.toHaveBeenCalled()

    unmount()
  })
})
