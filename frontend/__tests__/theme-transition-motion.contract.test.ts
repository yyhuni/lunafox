import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const globals = readFileSync(path.resolve(process.cwd(), "app/globals.css"), "utf8")
const globeSource = readFileSync(path.resolve(process.cwd(), "components/overview/agent-globe-card.tsx"), "utf8")
const themeProviderSource = readFileSync(path.resolve(process.cwd(), "components/providers/theme-provider.tsx"), "utf8")
const headerSource = readFileSync(path.resolve(process.cwd(), "components/unified-header.tsx"), "utf8")

describe("theme transition motion contract", () => {
  it("keeps explicit theme changes immediate without page-level transition motion", () => {
    expect(globals).toContain("html[data-theme-sync]")
    expect(globals).toContain("transition: none !important")
    expect(globals).toContain("transition-duration: 0ms !important")
    expect(globals).not.toContain("--lunafox-theme-transition-duration")
    expect(globals).not.toContain("--lunafox-theme-view-transition-duration")
    expect(globals).not.toContain("transition-duration: var(--lunafox-theme-transition-duration)")
    expect(globals).not.toContain("::view-transition-old(root)")
    expect(globals).not.toContain("::view-transition-new(root)")
    expect(globals).not.toContain("@keyframes lunafox-theme-fade-out")
    expect(globals).not.toContain("@keyframes lunafox-theme-fade-in")
    expect(globals).not.toContain("transition: all")
  })

  it("keeps global CSS free of remote font imports before first paint", () => {
    expect(globals).not.toContain("fonts.googleapis")
    expect(globals).not.toContain("@import url(")
    expect(globals).not.toContain('"Google Sans"')
    expect(globals).not.toContain("@xyflow/react/dist/style.css")
    expect(globals).not.toContain("@xterm/xterm/css/xterm.css")
    expect(globals).toContain("font-family: var(--font-sans);")
  })

  it("applies explicit theme changes directly without browser view-transition state", () => {
    expect(themeProviderSource).toContain("applyThemeWithoutTransitions")
    expect(themeProviderSource).toContain('root.setAttribute("data-theme-sync", "true")')
    expect(themeProviderSource).toContain('root.removeAttribute("data-theme-sync")')
    expect(themeProviderSource).toContain("requestAnimationFrame")
    expect(themeProviderSource).not.toContain("startViewTransition")
    expect(themeProviderSource).not.toContain("flushSync")
    expect(themeProviderSource).not.toContain("data-theme-transition")
    expect(themeProviderSource).not.toContain('matchMedia("(prefers-reduced-motion: reduce)")')
    expect(themeProviderSource).toContain("syncDocumentTheme(resolvedTheme, nextSelection.mode)")
  })

  it("uses the shared appearance menu without stale direct-toggle motion selectors", () => {
    expect(globals).not.toContain(".lunafox-theme-toggle")
    expect(globals).not.toContain("--lunafox-theme-toggle-icon-duration")
    expect(globals).not.toContain("data-theme-state=\"dark\"")
    expect(headerSource).toContain("HeaderIconActionMenu")
    expect(headerSource).toContain("DropdownMenuRadioGroup")
    expect(headerSource).toContain("DropdownMenuRadioItem")
    expect(headerSource).toContain("themeModeOptions")
    expect(headerSource).toContain("IconCircleHalf2")
    expect(headerSource).toContain("IconSun")
    expect(headerSource).toContain("IconMoon")
    expect(globals).toContain(".lunafox-cobe-canvas")
    expect(globals).toContain("@keyframes lunafox-cobe-theme-enter")
    expect(globals).toContain("@media (prefers-reduced-motion: reduce)")
    expect(globals).toContain(".lunafox-cobe-canvas")
  })

  it("recreates the Cobe instance from registered theme metadata without remounting the canvas", () => {
    expect(globeSource).toContain("useColorTheme")
    expect(globeSource).toContain("currentTheme")
    expect(globeSource).toContain("buildThemeSyncedCobeGlobeTheme(theme)")
    expect(globeSource).toContain("[theme]")
    expect(globeSource).toContain("data-cobe-theme={theme.id}")
    expect(globeSource).toContain("lunafox-cobe-canvas")
    expect(globeSource).not.toContain("[isDarkTheme]")
    expect(globeSource).not.toContain('data-cobe-theme={isDarkTheme ? "pulse" : "light"}')
    expect(globeSource).not.toContain('key={theme.id}')
  })
})
