import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/unified-header.tsx"), "utf8")

describe("unified-header contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function UnifiedHeader")
    expect(source).toContain("className")
    expect(source).toContain("from \"next/dynamic\"")
  })

  it("uses semantic success utility for the shell status dot", () => {
    expect(source).not.toContain("getStatusToneBgClass")
    expect(source).not.toContain("bg-[var(--success)]")
    expect(source).not.toContain('className="bg-success h-1.5 rounded-full w-1.5"')
  })

  it("does not render the product brand block in the top bar", () => {
    expect(source).not.toContain("LunaFox Logo")
    expect(source).not.toContain("logoSrc")
    expect(source).not.toContain("md:w-(--sidebar-width)")
  })

  it("does not expose a support-author CTA in the top bar", () => {
    expect(source).not.toContain('aria-label={t("supportAuthor")}')
    expect(source).not.toContain('/settings/support/')
    expect(source).not.toContain("IconHeart")
  })

  it("exposes a shadcn-style GitHub star action in the header action row", () => {
    expect(source).toContain('from "@/components/github-star-button"')
    expect(source).toContain('from "@/components/ui/separator"')
    expect(source).toContain("const HeaderActionSeparator")
    expect(source).toContain("<GithubStarButton />")
    expect(source).toContain('<Separator orientation="vertical" className="mx-2 h-4! w-px self-center bg-muted-foreground/35"')
    expect(source).toContain("Array.from({ length: 7 })")

    const githubIndex = source.indexOf("<GithubStarButton />")

    expect(githubIndex).toBeGreaterThan(-1)
  })

  it("exposes server resource usage from the header action row", () => {
    expect(source).toContain('from "@/components/server-resource-popover"')
    expect(source).toContain("<ServerResourcePopover />")

    const quickScanIndex = source.indexOf("<QuickScanHeaderTrigger />")
    const resourceIndex = source.indexOf("<ServerResourcePopover />")
    const notificationIndex = source.indexOf("<NotificationDrawer")

    expect(resourceIndex).toBeGreaterThan(quickScanIndex)
    expect(resourceIndex).toBeLessThan(notificationIndex)
  })

  it("exposes MCP access as a compact top-bar popover", () => {
    expect(source).toContain('from "@/components/mcp-access-popover"')
    expect(source).toContain("<McpAccessPopover />")

    const notificationIndex = source.indexOf("<NotificationDrawer")
    const mcpIndex = source.indexOf("<McpAccessPopover />")
    const languageIndex = source.indexOf("<LanguageSwitcher />")

    expect(mcpIndex).toBeGreaterThan(notificationIndex)
    expect(mcpIndex).toBeLessThan(languageIndex)
    expect(source).toContain("Array.from({ length: 7 })")
  })

  it("keeps GitHub as the far-right project action behind one shadcn-style separator", () => {
    const firstSeparatorIndex = source.indexOf("<HeaderActionSeparator />")
    const githubIndex = source.indexOf("<GithubStarButton />")
    const secondSeparatorIndex = source.indexOf("<HeaderActionSeparator />", firstSeparatorIndex + 1)

    expect(firstSeparatorIndex).toBeGreaterThan(-1)
    expect(firstSeparatorIndex).toBeLessThan(githubIndex)
    expect(secondSeparatorIndex).toBe(-1)
  })

  it("uses the shared appearance menu as the only theme control", () => {
    expect(source).toContain("useColorTheme")
    expect(source).toContain('from "@/components/shared/dropdown-menu-owners"')
    expect(source).toContain("HeaderIconActionMenu")
    expect(source).toContain("DropdownMenuRadioGroup")
    expect(source).toContain("DropdownMenuRadioItem")
    expect(source).toContain("themeModeOptions")
    expect(source).toContain("themeMenuLabel")
    expect(source).toContain("handleThemeModeChange")
    expect(source).toContain("onValueChange={handleThemeModeChange}")
    expect(source).toContain("value={mode}")
    expect(source).toContain('t("themeModeSystem")')
    expect(source).toContain('t("themeModeLight")')
    expect(source).toContain('t("themeModeDark")')
    expect(source).toContain("IconCircleHalf2")
    expect(source).toContain("IconSun")
    expect(source).toContain("IconMoon")
  })

  it("summarizes the selected mode with the matching icon", () => {
    expect(source).toContain("activeThemeMode")
    expect(source).toContain("ActiveThemeModeIcon")
    expect(source).toContain("<ActiveThemeModeIcon className=\"h-4 w-4\"")
    expect(source).toContain("<HeaderIconActionMenu")
    expect(source).toContain("setMode(value as ThemeModeId)")
    expect(source).not.toContain("lunafox-theme-toggle")
  })

  it("places the sidebar trigger at the far left of the top bar", () => {
    const triggerIndex = source.indexOf("<SidebarTrigger")
    const quickScanIndex = source.indexOf("<QuickScanHeaderTrigger />")

    expect(triggerIndex).toBeGreaterThan(-1)
    expect(triggerIndex).toBeLessThan(quickScanIndex)
    expect(source).toContain('className="size-8"')
    expect(source).not.toContain('className="md:hidden"')
  })

  it("delegates quick scan to its shared header trigger while retaining the theme control", () => {
    expect(source).toContain("QuickScanHeaderTrigger")
    expect(source).toContain('import("@/components/scan/quick-scan-header-trigger")')
    expect(source).toContain("<QuickScanHeaderTrigger />")
    expect(source).not.toContain("QuickScanDialog")
    expect(source).not.toContain('data-slot="quick-scan-trigger"')
    expect(source).toContain('ariaLabel={themeMenuLabel}')
    expect(source).toContain('className="h-4 w-4"')
    expect(source).not.toContain('className="h-7 w-7"')
    expect(source).not.toContain('<span className="hidden sm:inline">{t("supportAuthor")}</span>')
  })

  it("keeps the right action cluster visually tight", () => {
    expect(source).toContain("gap-0.5")
    expect(source).toContain("md:gap-1")
    expect(source).not.toContain("md:gap-2 ml-auto")
  })
})
