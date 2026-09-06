import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/notifications/notification-drawer-state.ts"), "utf8")

describe("notification-drawer-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useNotificationDrawerState")
    expect(source).toContain("from \"react\"")
    expect(source).toContain("useSyncNotificationLocale")
    expect(source).toContain("useSyncNotificationLocale(locale)")
    expect(source).toContain("useMarkAllNotificationsRead")
    expect(source).toContain("handleMarkAll")
    expect(source).not.toContain("useBootstrapNotificationLocale")
    expect(source).not.toContain("navigator.language")
    expect(source).not.toContain("useMarkNotificationRead")
    expect(source).not.toContain("handleMarkRead")
  })
})
