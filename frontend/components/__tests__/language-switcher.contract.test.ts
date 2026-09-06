import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/language-switcher.tsx"), "utf8")

describe("language-switcher contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function LanguageSwitcher")
    expect(source).toContain("className")
    expect(source).toContain('from "next-intl"')
  })

  it("uses the shared compact icon action size for the header trigger", () => {
    expect(source).toContain('from "@/components/shared/dropdown-menu-owners"')
    expect(source).toContain("HeaderIconActionMenu")
    expect(source).toContain("DropdownMenuRadioGroup")
    expect(source).toContain("DropdownMenuRadioItem")
    expect(source).toContain("value={locale}")
    expect(source).not.toContain("DropdownMenuItem")
    expect(source).not.toContain("IconCheck")
    expect(source).toContain('className="h-4 w-4"')
    expect(source).not.toContain("DropdownMenuTrigger")
    expect(source).not.toContain("DropdownMenuContent")
  })

  it("persists locale preference through the shared locale cookie and refresh flow", () => {
    expect(source).toContain("LOCALE_COOKIE_NAME")
    expect(source).toContain("document.cookie")
    expect(source).toContain('from "@/hooks/use-notification-settings"')
    expect(source).toContain("useUpdateNotificationLocale")
    expect(source).toContain("await updateNotificationLocale.mutateAsync(newLocale)")
    expect(source).not.toContain('from "@/services/notification-settings.service"')
    expect(source).toContain("router.refresh()")
    expect(source).not.toContain("replaceWithRouteProgress")
    expect(source).not.toContain("locale: newLocale")
  })
})
