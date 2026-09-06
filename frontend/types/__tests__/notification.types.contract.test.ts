import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "types/notification.types.ts"), "utf8")

describe("notification.types contract", () => {
  it("models frozen durable inbox snapshots and token pagination", () => {
    expect(source).toContain("export const NOTIFICATION_KINDS")
    expect(source).toContain('"scan-succeeded"')
    expect(source).toContain('"agent-offline"')
    expect(source).toContain('"nuclei-poc-sync-succeeded"')
    expect(source).toContain('"nuclei-poc-sync-failed"')
    expect(source).toContain("EXTERNALLY_DELIVERABLE_NOTIFICATION_KINDS")
    expect(source).toContain("export interface NotificationInboxItem")
    expect(source).toContain("readAt?: string | null")
    expect(source).toContain("pageToken?: string")
    expect(source).not.toContain("asset-discovered")
    expect(source).not.toContain("low\" | \"normal")
  })
})
