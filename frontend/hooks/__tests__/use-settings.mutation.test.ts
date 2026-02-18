import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  apiKeySettingsKeys,
  useUpdateApiKeySettings,
} from "@/hooks/use-api-key-settings"
import {
  notificationSettingsKeys,
  useUpdateNotificationSettings,
} from "@/hooks/use-notification-settings"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const apiKeySettingsServiceMocks = vi.hoisted(() => ({
  getSettings: vi.fn(),
  updateSettings: vi.fn(),
}))

const notificationSettingsServiceMocks = vi.hoisted(() => ({
  getSettings: vi.fn(),
  updateSettings: vi.fn(),
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
      fofa: { enabled: true, email: "ops@acme.test", apiKey: "secret" },
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUpdateApiKeySettings(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        fofa: { enabled: true, email: "ops@acme.test", apiKey: "secret" },
      })
    })

    expect(toastMocks.success).toHaveBeenCalledWith("toast.apiKeys.settings.success")
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: apiKeySettingsKeys.settings,
    })
  })

  it("更新通知设置成功时保留成功提示与 query 失效", async () => {
    notificationSettingsServiceMocks.updateSettings.mockResolvedValue({
      discord: {
        enabled: true,
        webhookUrl: "https://discord.test/webhook",
      },
      wecom: {
        enabled: false,
        webhookUrl: "",
      },
      categories: {
        scan: true,
        vulnerability: true,
        asset: true,
        system: false,
      },
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUpdateNotificationSettings(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        discord: {
          enabled: true,
          webhookUrl: "https://discord.test/webhook",
        },
        wecom: {
          enabled: false,
          webhookUrl: "",
        },
        categories: {
          scan: true,
          vulnerability: true,
          asset: true,
          system: false,
        },
      })
    })

    expect(toastMocks.success).toHaveBeenCalledWith("toast.notification.settings.success")
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: notificationSettingsKeys.settings,
    })
  })

  it("更新通知设置失败时保留错误码映射与 fallback key", async () => {
    notificationSettingsServiceMocks.updateSettings.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useUpdateNotificationSettings())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          discord: {
            enabled: true,
            webhookUrl: "https://discord.test/webhook",
          },
          wecom: {
            enabled: false,
            webhookUrl: "",
          },
          categories: {
            scan: true,
            vulnerability: true,
            asset: false,
            system: true,
          },
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.notification.settings.error"
    )
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
          fofa: { enabled: true, email: "ops@acme.test", apiKey: "bad-key" },
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "INVALID_API_KEY",
      "toast.apiKeys.settings.error"
    )
  })
})
