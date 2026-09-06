import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  apiKeySettingsKeys,
  useUpdateApiKeySettings,
} from "@/hooks/use-api-key-settings"
import {
  useUpdateNotificationDestination,
} from "@/hooks/use-notification-settings"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const apiKeySettingsServiceMocks = vi.hoisted(() => ({
  getSettings: vi.fn(),
  updateSettings: vi.fn(),
}))

const notificationSettingsServiceMocks = vi.hoisted(() => ({
  updateDestination: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/api-key-settings.service", () => ({
  ApiKeySettingsService: apiKeySettingsServiceMocks,
}))

vi.mock("@/services/notification-settings.service", () => ({
  NotificationSettingsService: notificationSettingsServiceMocks,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

describe("use-settings mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("更新 API Key 设置成功时保留成功提示与 query 失效", async () => {
    apiKeySettingsServiceMocks.updateSettings.mockResolvedValue({
      providers: {},
      definitions: [],
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUpdateApiKeySettings(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        providers: {
          fofa: { enabled: true, values: { email: "ops@acme.test", apiKey: "secret" } },
        },
      })
    })

    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.apiKeys.settings.success",
      undefined,
      "update-api-key-settings"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: apiKeySettingsKeys.settings,
    })
  })

  it("保存单个推送目标时只提交服务请求，由页面协调器负责反馈与基线", async () => {
    notificationSettingsServiceMocks.updateDestination.mockResolvedValue({
      provider: "discord",
      credential: "https://discord.test/webhook",
      enabled: true,
      subscriptions: ["scan-failed"],
    })

    const queryClient = createTestQueryClient()
    const { result } = renderHookWithProviders(() => useUpdateNotificationDestination(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        provider: "discord",
        data: {
          credential: "https://discord.test/webhook",
          enabled: true,
          subscriptions: ["scan-failed"],
        },
      })
    })

    expect(notificationSettingsServiceMocks.updateDestination).toHaveBeenCalledWith("discord", {
      credential: "https://discord.test/webhook",
      enabled: true,
      subscriptions: ["scan-failed"],
    })
    expect(toastMocks.success).not.toHaveBeenCalled()
  })

  it("更新 API Key 设置失败时保留错误码映射与 fallback key", async () => {
    apiKeySettingsServiceMocks.updateSettings.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "INVALID_API_KEY",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useUpdateApiKeySettings())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          providers: {
            fofa: { enabled: true, values: { email: "ops@acme.test", apiKey: "bad-key" } },
          },
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "INVALID_API_KEY",
      "toast.apiKeys.settings.error",
      "update-api-key-settings"
    )
  })
})
