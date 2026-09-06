import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

import {
  COLOR_THEMES,
  THEME_MODES,
  DEFAULT_DARK_COLOR_THEME_ID,
  DEFAULT_COLOR_THEME_ID,
  DEFAULT_LIGHT_COLOR_THEME_ID,
  DEFAULT_THEME_MODE_ID,
  deriveColorThemeSelection,
  getColorTheme,
  isColorThemeId,
  isThemeModeId,
  resolveColorThemeIdForSelection,
  resolveColorTheme,
  resolveColorThemeId,
  resolveStoredColorThemeId,
  resolveStoredColorThemeSelection,
  resolveThemeModeId,
} from "@/lib/color-themes"

const lunafoxDarkCss = readFileSync(
  path.resolve(process.cwd(), "styles/themes/lunafox-dark.css"),
  "utf8"
)
const lunafoxLightCss = readFileSync(
  path.resolve(process.cwd(), "styles/themes/lunafox-light.css"),
  "utf8"
)
const themeIndexCss = readFileSync(
  path.resolve(process.cwd(), "styles/themes/index.css"),
  "utf8"
)

const lunafoxDarkTokenBlock = lunafoxDarkCss.match(
  /\[data-theme="lunafox-dark"\]\s*\{([\s\S]*?)\n\}/
)?.[1]
const lunafoxLightTokenBlock = lunafoxLightCss.match(
  /\[data-theme="lunafox-light"\]\s*\{([\s\S]*?)\n\}/
)?.[1]

const LEGACY_DEFAULT_THEME_STORAGE_VALUE = "shad" + "cn-default"
const REFERENCE_NEUTRAL_DARK_BASELINE_TOKENS: Record<string, string> = {
  "--background": "oklch(0.145 0 0)",
  "--foreground": "oklch(0.985 0 0)",
  "--card": "oklch(0.205 0 0)",
  "--card-foreground": "oklch(0.985 0 0)",
  "--popover": "oklch(0.205 0 0)",
  "--popover-foreground": "oklch(0.985 0 0)",
  "--primary": "oklch(0.922 0 0)",
  "--primary-foreground": "oklch(0.205 0 0)",
  "--secondary": "oklch(0.269 0 0)",
  "--secondary-foreground": "oklch(0.985 0 0)",
  "--muted": "oklch(0.269 0 0)",
  "--muted-foreground": "oklch(0.708 0 0)",
  "--accent": "oklch(0.269 0 0)",
  "--accent-foreground": "oklch(0.985 0 0)",
  "--destructive": "oklch(0.704 0.191 22.216)",
  "--border": "oklch(1 0 0 / 10%)",
  "--input": "oklch(1 0 0 / 15%)",
  "--chart-1": "oklch(0.488 0.243 264.376)",
  "--chart-2": "oklch(0.696 0.17 162.48)",
  "--chart-3": "oklch(0.769 0.188 70.08)",
  "--chart-4": "oklch(0.627 0.265 303.9)",
  "--chart-5": "oklch(0.645 0.246 16.439)",
  "--sidebar": "oklch(0.205 0 0)",
  "--sidebar-foreground": "oklch(0.985 0 0)",
  "--sidebar-accent": "oklch(0.269 0 0)",
  "--sidebar-accent-foreground": "oklch(0.985 0 0)",
  "--sidebar-border": "oklch(1 0 0 / 10%)",
}

function getThemeTokenValue(token: string): string | undefined {
  return lunafoxDarkTokenBlock?.match(new RegExp(`${token}:\\s*([^;]+);`))?.[1]
}

const REQUIRED_SEVERITY_TOKENS = [
  "--severity-critical",
  "--severity-critical-background",
  "--severity-critical-border",
  "--severity-critical-hover",
  "--severity-high",
  "--severity-high-background",
  "--severity-high-border",
  "--severity-high-hover",
  "--severity-medium",
  "--severity-medium-background",
  "--severity-medium-border",
  "--severity-medium-hover",
  "--severity-low",
  "--severity-low-background",
  "--severity-low-border",
  "--severity-low-hover",
  "--severity-info",
  "--severity-info-background",
  "--severity-info-border",
  "--severity-info-hover",
]

describe("color-themes contract", () => {
  it("keeps themes in a flat registry with default light and dark entries", () => {
    const themeIds = COLOR_THEMES.map((theme) => theme.id)

    expect(DEFAULT_COLOR_THEME_ID).toBe(DEFAULT_LIGHT_COLOR_THEME_ID)
    expect(DEFAULT_LIGHT_COLOR_THEME_ID).toBe("lunafox-light")
    expect(DEFAULT_DARK_COLOR_THEME_ID).toBe("lunafox-dark")
    expect(themeIds).toContain(DEFAULT_LIGHT_COLOR_THEME_ID)
    expect(themeIds).toContain(DEFAULT_DARK_COLOR_THEME_ID)
    expect(themeIds).toEqual(["lunafox-light", "lunafox-dark"])
    expect(new Set(themeIds).size).toBe(themeIds.length)
    expect(COLOR_THEMES.every((theme) => typeof theme.isDark === "boolean")).toBe(true)
  })

  it("keeps appearance mode bound to the default light/dark pair", () => {
    expect(DEFAULT_THEME_MODE_ID).toBe("system")
    expect(THEME_MODES.map((mode) => mode.id)).toEqual(["system", "light", "dark"])
    expect(resolveThemeModeId(undefined)).toBe("system")
    expect(resolveColorThemeIdForSelection({ mode: "light" }, false)).toBe("lunafox-light")
    expect(resolveColorThemeIdForSelection({ mode: "dark" }, false)).toBe("lunafox-dark")
    expect(resolveColorThemeIdForSelection({ mode: "system" }, true)).toBe("lunafox-dark")
    expect(resolveColorThemeIdForSelection({ mode: "system" }, false)).toBe("lunafox-light")
  })

  it("falls back to the default theme for unknown ids", () => {
    expect(resolveColorThemeId("unknown-theme")).toBe(DEFAULT_COLOR_THEME_ID)
    expect(resolveColorTheme("unknown-theme").id).toBe(DEFAULT_COLOR_THEME_ID)
  })

  it("resolves registered theme metadata", () => {
    expect(isColorThemeId("bauhaus")).toBe(false)
    expect(isColorThemeId("lunafox-light")).toBe(true)
    expect(isColorThemeId("lunafox-dark")).toBe(true)
    expect(isThemeModeId("system")).toBe(true)
    expect(isColorThemeId(LEGACY_DEFAULT_THEME_STORAGE_VALUE)).toBe(false)
    expect(getColorTheme("lunafox-light").name).toBe("LunaFox Light")
    expect(getColorTheme("lunafox-light").description).toContain("signal-led accents")
    expect(getColorTheme("lunafox-dark").isDark).toBe(true)
    expect(getColorTheme("lunafox-dark").description).toContain("Neutral dark baseline")
    expect(getColorTheme("lunafox-dark").description).toContain("focused data cockpit")
  })

  it("maps legacy and removed storage values to the default theme pair", () => {
    expect(resolveStoredColorThemeId("bauhaus")).toBe(DEFAULT_COLOR_THEME_ID)
    expect(resolveStoredColorThemeId(LEGACY_DEFAULT_THEME_STORAGE_VALUE)).toBe(DEFAULT_COLOR_THEME_ID)
    for (const themeId of [
      "cherry-cocoa-light", "rose-cocoa-light", "graphite-contrast-light", "mist-teal-light",
      "ink-navy-light", "paper-ink-light", "blue-light", "smoked-amethyst-light",
    ]) {
      expect(resolveStoredColorThemeId(themeId)).toBe(DEFAULT_LIGHT_COLOR_THEME_ID)
    }
    for (const themeId of [
      "cherry-cocoa-dark", "rose-cocoa-dark", "graphite-contrast-dark", "mist-teal-dark",
      "ink-navy-dark", "paper-ink-dark", "blue-dark", "smoked-amethyst-dark",
    ]) {
      expect(resolveStoredColorThemeId(themeId)).toBe(DEFAULT_DARK_COLOR_THEME_ID)
    }
  })

  it("resolves final runtime themes from appearance mode only", () => {
    expect(resolveColorThemeIdForSelection({ mode: "light" }, false)).toBe("lunafox-light")
    expect(resolveColorThemeIdForSelection({ mode: "dark" }, false)).toBe("lunafox-dark")
    expect(resolveColorThemeIdForSelection({ mode: "system" }, true)).toBe("lunafox-dark")
    expect(resolveColorThemeIdForSelection({ mode: "system" }, false)).toBe("lunafox-light")
  })

  it("derives appearance mode from the default runtime themes", () => {
    expect(deriveColorThemeSelection("lunafox-light")).toEqual({ mode: "light" })
    expect(deriveColorThemeSelection("lunafox-dark")).toEqual({ mode: "dark" })
    expect(resolveStoredColorThemeSelection({
      storedMode: "dark",
      storedTheme: "lunafox-light",
    })).toEqual({ mode: "dark" })
    expect(resolveStoredColorThemeSelection({
      storedTheme: "rose-cocoa-light",
    })).toEqual({ mode: "light" })
    expect(resolveStoredColorThemeSelection({
      storedTheme: LEGACY_DEFAULT_THEME_STORAGE_VALUE,
    })).toEqual({ mode: "light" })
    expect(resolveStoredColorThemeSelection({
      storedMode: "dark",
      storedTheme: "lunafox-light",
    })).toEqual({ mode: "dark" })
    expect(resolveStoredColorThemeSelection({
      storedMode: "dark",
      storedTheme: "paper-ink-dark",
    })).toEqual({ mode: "dark" })
    expect(resolveStoredColorThemeSelection({
      storedTheme: "blue-dark",
    })).toEqual({ mode: "dark" })
    expect(resolveStoredColorThemeSelection({})).toEqual({ mode: "system" })
  })

  it("keeps lunafox-dark aligned to the shared neutral dark token baseline", () => {
    expect(lunafoxDarkTokenBlock).toBeDefined()

    for (const [token, expectedValue] of Object.entries(REFERENCE_NEUTRAL_DARK_BASELINE_TOKENS)) {
      expect(getThemeTokenValue(token)).toBe(expectedValue)
    }
  })

  it("keeps LunaFox operational semantic state tokens as dark-theme extensions", () => {
    for (const token of ["--success", "--warning", "--error", "--info"]) {
      expect(getThemeTokenValue(token)).toMatch(/^oklch\(/)
    }
  })

  it("aligns interaction feedback to the indigo accent token", () => {
    for (const tokenBlock of [lunafoxLightTokenBlock, lunafoxDarkTokenBlock]) {
      expect(tokenBlock).toContain("--ring: var(--interaction-accent);")
      expect(tokenBlock).toContain("--sidebar-primary: var(--interaction-accent);")
      expect(tokenBlock).toContain("--sidebar-primary-foreground: var(--interaction-accent-foreground);")
      expect(tokenBlock).toContain("--sidebar-ring: var(--interaction-accent);")
      expect(tokenBlock).toContain("--highlight: var(--interaction-accent);")
    }
  })

  it("keeps theme metadata token-backed instead of raw color literals", () => {
    const rawColorLiteral = /#[0-9A-Fa-f]{3,8}\b/

    for (const theme of COLOR_THEMES) {
      expect(theme.color).toMatch(/^var\(--/)
      expect(theme.colors.every((color) => color.startsWith("var(--"))).toBe(true)
    }

    expect(readFileSync(path.resolve(process.cwd(), "lib/color-themes.ts"), "utf8")).not.toMatch(rawColorLiteral)
  })

  it("registers only the default light/dark stylesheets with complete theme tokens", () => {
    expect(themeIndexCss).toContain('@import "./base.css";')
    expect(themeIndexCss).toContain('@import "./lunafox-light.css";')
    expect(themeIndexCss).toContain('@import "./lunafox-dark.css";')

    for (const token of [
      "--background",
      "--foreground",
      "--card",
      "--popover",
      "--primary",
      "--secondary",
      "--muted",
      "--accent",
      "--destructive",
      "--border",
      "--input",
      "--ring",
      "--chart-1",
      "--sidebar",
      "--highlight",
      "--pixel-blast-color",
      "--success",
      "--warning",
      "--error",
      "--info",
      "--scrollbar-thumb",
      "--quick-scan-flash-dim",
      "--button-glow-strong",
      "--splash-primary",
      "--brand-discord",
      "--brand-wecom",
      "--brand-feishu",
      "--brand-github-star",
    ]) {
      expect(lunafoxLightTokenBlock).toContain(`${token}:`)
      expect(lunafoxDarkTokenBlock).toContain(`${token}:`)
    }
  })

  it("keeps severity colors owned by every runtime theme", () => {
    const tokenBlocks = [
      lunafoxLightTokenBlock,
      lunafoxDarkTokenBlock,
    ]

    for (const block of tokenBlocks) {
      expect(block).toBeDefined()
      for (const token of REQUIRED_SEVERITY_TOKENS) {
        expect(block).toContain(`${token}:`)
      }
    }

    expect(themeIndexCss).toContain('@import "./lunafox-light.css";')
    expect(themeIndexCss).toContain('@import "./lunafox-dark.css";')
  })
})
