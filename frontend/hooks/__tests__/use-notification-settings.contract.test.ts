import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-notification-settings.ts"), "utf8")

describe("use-notification-settings contract", () => {
  it("exposes only destination and locale hooks", () => {
    expect(source).toContain("export function useNotificationDestinations")
    expect(source).toContain("export function useUpdateNotificationDestination")
    expect(source).toContain("export function useTestNotificationDestination")
    expect(source).toContain("export function useUpdateNotificationLocale")
    expect(source).toContain("export function useSyncNotificationLocale")
    expect(source).not.toContain("navigator.language")
    expect(source).not.toContain("browserLocale")
    expect(source).toContain('from "@tanstack/react-query"')
    expect(source).not.toContain("useNotificationPreferences")
    expect(source).not.toContain("useUpdateNotificationPreferences")
    expect(source).not.toContain("useNotificationSettings")
  })
})
