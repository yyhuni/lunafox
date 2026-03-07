import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { transformBackendNotification } from "@/hooks/use-notification-sse"

describe("use-notification-sse transform", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date("2026-03-06T12:00:00Z"))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it("只读取 camelCase 通知字段", () => {
    const notification = transformBackendNotification({
      id: 1,
      title: "Scan complete",
      message: "task finished",
      level: "low",
      createdAt: "2026-01-01T00:00:00.000Z",
      isRead: true,
    })

    expect(notification.createdAt).toBe("2026-01-01T00:00:00.000Z")
    expect(notification.unread).toBe(false)
  })
})
