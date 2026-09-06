import { act, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  notificationKeys,
  useMarkAllNotificationsRead,
  useNotifications,
} from "@/hooks/use-notifications"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const notificationServiceMocks = vi.hoisted(() => ({
  list: vi.fn(),
  markAllRead: vi.fn(),
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

  it("禁用 inbox 查询时不请求服务端", async () => {
    const { result } = renderHookWithProviders(
      () => useNotifications({ pageSize: 100 }, { enabled: false })
    )

    await waitFor(() => {
      expect(result.current.fetchStatus).toBe("idle")
    })

    expect(notificationServiceMocks.list).not.toHaveBeenCalled()
  })

  it("全部标记已读的未知失败会回读持久 inbox 且不重放命令", async () => {
    notificationServiceMocks.markAllRead.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "RATE_LIMITED",
          },
        },
      },
    })

    const queryClient = createTestQueryClient()
    const refetchSpy = vi.spyOn(queryClient, "refetchQueries")
    const { result } = renderHookWithProviders(() => useMarkAllNotificationsRead(), { queryClient })

    await act(async () => {
      await expect(result.current.mutateAsync(undefined)).rejects.toBeDefined()
    })

    expect(notificationServiceMocks.markAllRead).toHaveBeenCalledTimes(1)
    expect(refetchSpy).toHaveBeenCalledWith({
      queryKey: notificationKeys.all,
      type: "active",
    })
    expect(toastMocks.errorFromCode).not.toHaveBeenCalled()
  })
})
