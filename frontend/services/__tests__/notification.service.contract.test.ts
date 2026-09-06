import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "services/notification.service.ts"), "utf8")

describe("notification.service contract", () => {
  it("uses current-user resources, a bulk read method, and fetch SSE", () => {
    expect(source).toContain('from "@/lib/api-client"')
    expect(source).toContain('const inboxPath = "/users/current/notifications"')
    expect(source).toContain(":markAllRead")
    expect(source).not.toContain(":markRead")
    expect(source).not.toContain("static async markRead")
    expect(source).toContain("getStreamToken")
    expect(source).toContain("getFreshStreamToken")
    expect(source).toContain("ensurePrimaryTokenFresh")
    expect(source).toContain("renewStreamToken")
    expect(source).toContain("openStream")
    expect(source).toContain("Authorization: `Bearer ${token}`")
    expect(source).not.toContain("/notifications/mark-all-as-read")
  })
})
