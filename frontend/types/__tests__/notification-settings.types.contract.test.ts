import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "types/notification-settings.types.ts"), "utf8")

describe("notification-settings.types contract", () => {
  it("models independent Discord, WeCom, and Feishu destinations in one fixed order", () => {
    expect(source).toContain('export type NotificationDestinationProvider = "discord" | "wecom" | "feishu"')
    expect(source).toContain('export const NOTIFICATION_DESTINATION_PROVIDERS = ["discord", "wecom", "feishu"]')
    expect(source).toContain("enabled: boolean")
    expect(source).not.toContain("NotificationPreferences")
    expect(source).not.toContain("scanEnabled")
    expect(source).not.toContain("vulnerabilityEnabled")
    expect(source).not.toContain("systemEnabled")
    expect(source).toContain("export interface NotificationDestination")
    expect(source).toContain("subscriptions: ExternalNotificationKind[]")
    expect(source).toContain("supportedKinds: ExternalNotificationKind[]")
    expect(source).toContain("requiresWebhookUpdate: boolean")
    expect(source).toContain("export interface TestNotificationDestinationResponse")
    expect(source).not.toContain("DiscordSettings")
    expect(source).not.toContain("EmailSettings")
  })
})
