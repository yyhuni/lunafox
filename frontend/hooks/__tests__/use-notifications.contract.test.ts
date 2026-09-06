import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-notifications.ts"), "utf8")

describe("use-notifications contract", () => {
  it("uses durable inbox pagination and the bulk read command only", () => {
    expect(source).toContain("export function useNotifications")
    expect(source).toContain("export function useNotificationPages")
    expect(source).toContain("export function useMarkAllNotificationsRead")
    expect(source).toContain("NotificationService.markAllRead")
    expect(source).toContain('from "@tanstack/react-query"')
    expect(source).not.toContain("useMarkNotificationRead")
    expect(source).not.toContain("NotificationService.markRead")
    expect(source).not.toContain("useNotificationWebSocket")
  })
})
