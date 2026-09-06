import { afterEach, describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

import {
  getMockNotificationDestinations,
  getMockNotificationTestDeliveryUnavailableResult,
  resetMockNotificationSettings,
  updateMockNotificationDestination,
} from "../notification-settings"

const source = readFileSync(path.resolve(process.cwd(), "mock/data/notification-settings.ts"), "utf8")
const handlerSource = readFileSync(path.resolve(process.cwd(), "mock/handlers/index.ts"), "utf8")

describe("notification-settings contract", () => {
  afterEach(() => {
    resetMockNotificationSettings()
  })

  it("mirrors the fixed exact-kind destination contracts without inbox preferences", () => {
    expect(source).toContain("export function getMockNotificationDestinations")
    expect(source).toContain("export function updateMockNotificationDestination")
    expect(source).toContain("getMockNotificationTestDeliveryUnavailableResult")
    expect(source).toContain("enabled notification destination requires a subscription")
    expect(source).toContain("isOfficialWebhookURL")
    expect(source).toContain("provider's official HTTPS webhook URL")
    expect(source).toContain('from "@/types/notification-settings.types"')
    expect(source).toContain("NOTIFICATION_DESTINATION_PROVIDERS")
    expect(source).toContain("createMockNotificationDestinations")
    expect(handlerSource).toContain("NOTIFICATION_DESTINATION_PROVIDERS.includes(provider)")
    expect(source).not.toContain("NotificationPreferences")
    expect(source).not.toContain("notificationPreferences")
    expect(source).not.toContain("bootstrapMockNotificationLocale")
    expect(source).not.toContain("getMockNotificationSettings")
    expect(handlerSource).not.toContain("notificationLocale:bootstrap")
    expect(handlerSource).not.toContain("browserLocale")
  })

  it("keeps Nuclei inbox-only kinds out of external destination choices", () => {
    expect(source).toContain("EXTERNALLY_DELIVERABLE_NOTIFICATION_KINDS")
    expect(source).not.toContain("supportedKinds: [...NOTIFICATION_KINDS]")
    expect(source).not.toContain("nuclei-poc-sync-succeeded")
    expect(source).not.toContain("nuclei-poc-sync-failed")
  })

  it("uses the shared fixed order and updates one external destination without changing the others", () => {
    expect(getMockNotificationDestinations().results.map(({ provider }) => provider)).toEqual(["discord", "wecom", "feishu"])

    expect(updateMockNotificationDestination("wecom", {
      credential: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=wecom-key",
      enabled: true,
      subscriptions: ["scan-failed"],
    })).toMatchObject({
      provider: "wecom",
      enabled: true,
      subscriptions: ["scan-failed"],
    })
    expect(getMockNotificationDestinations().results.find(({ provider }) => provider === "discord")).toMatchObject({
      enabled: true,
      subscriptions: ["scan-failed", "vulnerability-observed"],
    })
    expect(getMockNotificationDestinations().results.find(({ provider }) => provider === "feishu")).toMatchObject({
      credential: "",
      enabled: false,
      subscriptions: [],
    })
  })

  it("rejects credentials that would not pass the Server-owned provider boundary", () => {
    expect(() => updateMockNotificationDestination("discord", {
      credential: "https://discord.example.test/api/webhooks/mock-id/mock-token",
      enabled: true,
      subscriptions: ["scan-failed"],
    })).toThrow("official HTTPS webhook URL")

    expect(() => updateMockNotificationDestination("feishu", {
      credential: "https://open.feishu.cn/open-apis/bot/v2/hook/mock-token/",
      enabled: true,
      subscriptions: ["scan-failed"],
    })).toThrow("official HTTPS webhook URL")

    expect(() => updateMockNotificationDestination("feishu", {
      credential: " https://open.feishu.cn/open-apis/bot/v2/hook/mock-token ",
      enabled: true,
      subscriptions: ["scan-failed"],
    })).toThrow("official HTTPS webhook URL")

    expect(() => updateMockNotificationDestination("feishu", {
      credential: "https://open.feishu.cn/open-apis/bot/v2/hook/mock-token?",
      enabled: true,
      subscriptions: ["scan-failed"],
    })).toThrow("official HTTPS webhook URL")

    expect(updateMockNotificationDestination("feishu", {
      credential: "https://open.feishu.cn/open-apis/bot/v2/hook/mock-token",
      enabled: true,
      subscriptions: ["scan-failed"],
    })).toMatchObject({ provider: "feishu", enabled: true })
  })

  it("models test delivery as unavailable without changing a destination draft", () => {
    expect(getMockNotificationTestDeliveryUnavailableResult()).toEqual({ result: "unavailable" })
  })
})
