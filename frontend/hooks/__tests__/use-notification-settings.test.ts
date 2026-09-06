import { waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { useSyncNotificationLocale } from "@/hooks/use-notification-settings"

const serviceMocks = vi.hoisted(() => ({
  shouldSynchronizePageLocale: vi.fn(),
  updateLocale: vi.fn(),
  updateLocaleForPageChange: vi.fn(),
}))

vi.mock("@/services/notification-settings.service", () => ({
  NotificationSettingsService: serviceMocks,
}))

describe("useSyncNotificationLocale", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    serviceMocks.shouldSynchronizePageLocale.mockReturnValue(true)
    serviceMocks.updateLocale.mockImplementation(async (locale: "zh" | "en") => ({ locale }))
  })

  it("同步页面 locale，而不是浏览器语言", async () => {
    Object.defineProperty(window.navigator, "language", {
      configurable: true,
      value: "en-US",
    })

    const { result, rerender } = renderHookWithProviders(
      ({ locale }: { locale: "zh" | "en" }) => useSyncNotificationLocale(locale),
      { initialProps: { locale: "zh" } },
    )

    await waitFor(() => expect(serviceMocks.updateLocale).toHaveBeenCalledWith("zh"))
    rerender({ locale: "zh" })
    expect(serviceMocks.updateLocale).toHaveBeenCalledTimes(1)
    expect(result.current.isSuccess).toBe(true)
  })

  it("locale 变化时重试，即使前一次同步失败", async () => {
    serviceMocks.updateLocale
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce({ locale: "en" })

    const { result, rerender } = renderHookWithProviders(
      ({ locale }: { locale: "zh" | "en" }) => useSyncNotificationLocale(locale),
      { initialProps: { locale: "zh" } },
    )

    await waitFor(() => expect(result.current.isError).toBe(true))
    rerender({ locale: "en" })
    await waitFor(() => expect(serviceMocks.updateLocale).toHaveBeenLastCalledWith("en"))
    expect(serviceMocks.updateLocale).toHaveBeenCalledTimes(2)
  })

  it("在显式切换尚未提交页面前跳过旧 locale 的自动同步", () => {
    serviceMocks.shouldSynchronizePageLocale.mockReturnValue(false)

    renderHookWithProviders(
      ({ locale }: { locale: "zh" | "en" }) => useSyncNotificationLocale(locale),
      { initialProps: { locale: "en" } },
    )

    expect(serviceMocks.shouldSynchronizePageLocale).toHaveBeenCalledWith("en")
    expect(serviceMocks.updateLocale).not.toHaveBeenCalled()
  })
})
