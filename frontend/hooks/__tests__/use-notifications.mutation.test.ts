import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { notificationKeys, useMarkAllAsRead } from "@/hooks/use-notifications"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const notificationServiceMocks = vi.hoisted(() => ({
  getNotifications: vi.fn(),
  markAllAsRead: vi.fn(),
  getUnreadCount: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/notification.service", () => ({
  NotificationService: notificationServiceMocks,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

describe("use-notifications mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("全部标记已读成功时会失效 notifications 缓存", async () => {
    notificationServiceMocks.markAllAsRead.mockResolvedValue({
      updated: 3,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useMarkAllAsRead(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(undefined)
    })

    expect(notificationServiceMocks.markAllAsRead).toHaveBeenCalled()
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: notificationKeys.all,
    })
  })

  it("全部标记已读失败时不触发默认 errorFromCode 处理", async () => {
    notificationServiceMocks.markAllAsRead.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "RATE_LIMITED",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useMarkAllAsRead())

    await act(async () => {
      await expect(result.current.mutateAsync(undefined)).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).not.toHaveBeenCalled()
    expect(toastMocks.error).not.toHaveBeenCalled()
    expect(toastMocks.success).not.toHaveBeenCalled()
  })
})
