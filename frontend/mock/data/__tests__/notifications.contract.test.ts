import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "mock/data/notifications.ts"), "utf8")

describe("notifications contract", () => {
  it("mirrors current-user inbox pagination and independent read commands", () => {
    expect(source).toContain("export function getMockNotifications")
    expect(source).toContain("export function markMockNotificationRead")
    expect(source).toContain("export function markAllMockNotificationsRead")
    expect(source).toContain("pageToken")
    expect(source).toContain('from "@/types/notification.types"')
    expect(source).toContain('kind: "nuclei-poc-sync-succeeded"')
    expect(source).toContain('kind: "nuclei-poc-sync-failed"')
    expect(source).not.toContain("repoUrl")
    expect(source).not.toContain("diagnostics")
    expect(source).not.toContain("asset-discovered")
  })
})
