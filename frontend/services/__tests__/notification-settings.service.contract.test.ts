import { beforeEach, describe, expect, it, vi } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const apiMocks = vi.hoisted(() => ({
  get: vi.fn(),
  patch: vi.fn(),
  post: vi.fn(),
}))

vi.mock("@/lib/api-client", () => ({ api: apiMocks }))

import { api } from "@/lib/api-client"
import { NotificationSettingsService } from "@/services/notification-settings.service"

const source = readFileSync(path.resolve(process.cwd(), "services/notification-settings.service.ts"), "utf8")

describe("notification-settings.service contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("uses only installation destinations and locale resources", () => {
    expect(source).toContain('from "@/lib/api-client"')
    expect(source).toContain('"/settings/notificationDestinations"')
    expect(source).toContain("testDestination")
    expect(source).toContain("`${destinationsPath}/${provider}:testDelivery`")
    expect(source).toContain('"/users/current/notificationLocale"')
    expect(source).toContain("updateLocale")
    expect(source).not.toContain("bootstrapLocale")
    expect(source).not.toContain("browserLocale")
    expect(source).not.toContain("notificationPreferences")
    expect(source).not.toContain("Preferences")
    expect(source).not.toContain("/settings/notifications")
  })

  it("serializes locale writes so the latest page choice cannot be overwritten by an older sync", async () => {
    let resolveFirst: ((value: { data: { locale: "en" } }) => void) | undefined
    apiMocks.patch.mockReturnValueOnce(new Promise((resolve) => {
      resolveFirst = resolve
    }))
    apiMocks.patch.mockResolvedValueOnce({ data: { locale: "zh" } })

    const first = NotificationSettingsService.updateLocale("en")
    const second = NotificationSettingsService.updateLocale("zh")

    await Promise.resolve()
    expect(api.patch).toHaveBeenCalledTimes(1)
    expect(api.patch).toHaveBeenNthCalledWith(1, "/users/current/notificationLocale", { locale: "en" })

    resolveFirst?.({ data: { locale: "en" } })
    await expect(first).resolves.toEqual({ locale: "en" })
    await expect(second).resolves.toEqual({ locale: "zh" })
    expect(api.patch).toHaveBeenCalledTimes(2)
    expect(api.patch).toHaveBeenNthCalledWith(2, "/users/current/notificationLocale", { locale: "zh" })
  })

  it("blocks a delayed old-page sync until the selected page locale has mounted", async () => {
    apiMocks.patch.mockResolvedValueOnce({ data: { locale: "zh" } })

    await expect(NotificationSettingsService.updateLocaleForPageChange("zh")).resolves.toEqual({ locale: "zh" })
    expect(NotificationSettingsService.shouldSynchronizePageLocale("en")).toBe(false)
    expect(NotificationSettingsService.shouldSynchronizePageLocale("zh")).toBe(true)
    expect(api.patch).toHaveBeenCalledWith("/users/current/notificationLocale", { locale: "zh" })
  })
})
