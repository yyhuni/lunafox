import { describe, expect, it } from "vitest"
import { existsSync, readFileSync } from "node:fs"
import path from "node:path"

const sourcePath = path.resolve(process.cwd(), "lib/notification-provider-config.ts")

describe("notification-provider-config contract", () => {
  it("exports shared provider brand owner helpers", () => {
    expect(existsSync(sourcePath)).toBe(true)
    const source = readFileSync(sourcePath, "utf8")
    expect(source).toContain("export type NotificationProviderBrand")
    expect(source).toContain("export function getNotificationProviderIconSurfaceClass")
    expect(source).toContain("export function getNotificationProviderIconClass")
    expect(source).toContain("var(--brand-discord)")
    expect(source).toContain("var(--brand-wecom)")
    expect(source).toContain("var(--brand-feishu)")
    expect(source).not.toContain("#5865F2")
    expect(source).not.toContain("#07C160")
  })
})
