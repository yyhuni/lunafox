import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const THEME_DIR = path.resolve(process.cwd(), "styles/themes")
const RUNTIME_THEME_FILES = [
  "lunafox-light.css",
  "lunafox-dark.css",
] as const

const SPLASH_ACCENT_TOKENS = [
  "--splash-primary",
  "--splash-primary-muted",
  "--splash-primary-faint",
  "--splash-secondary",
  "--splash-secondary-faint",
] as const

const PRODUCT_EFFECT_TOKENS = [
  "--pixel-blast-color",
] as const

const SUPPORT_CONFETTI_TOKENS = [
  "--support-confetti-1",
  "--support-confetti-2",
  "--support-confetti-3",
  "--support-confetti-4",
] as const

const COMMON_THEME_TOKENS = [
  "--radius",
  "--radius-control",
  "--radius-control-subtle",
  "--radius-surface",
  "--radius-overlay",
  "--radius-overlay-arrow",
  "--radius-badge",
  "--radius-pill",
  "--radius-round",
  "--overlay-backdrop-background",
  "--font-sans",
  "--font-mono",
  "--font-serif",
  "--tracking-normal",
  "--spacing",
  "--logo-background",
  "--progress-stripe",
  "--splash-scanline",
  "--support-scratch-text",
  "--support-scratch-text-muted",
  "--support-hologram-highlight",
  "--auth-boot-ring",
  "--auth-boot-ring-subtle",
  "--auth-boot-grid",
  "--auth-boot-progress-track",
  "--auth-boot-dot",
  "--auth-boot-dot-active",
  "--auth-terminal-glow",
  "--terminal-log-debug",
  "--terminal-log-warning",
] as const

const REQUIRED_RUNTIME_TOKENS = [
  "--background",
  "--foreground",
  "--card",
  "--card-foreground",
  "--popover",
  "--popover-foreground",
  "--primary",
  "--primary-foreground",
  "--interaction-accent",
  "--interaction-accent-foreground",
  "--secondary",
  "--secondary-foreground",
  "--muted",
  "--muted-foreground",
  "--accent",
  "--accent-foreground",
  "--destructive",
  "--destructive-foreground",
  "--border",
  "--input",
  "--ring",
  "--chart-1",
  "--chart-2",
  "--chart-3",
  "--chart-4",
  "--chart-5",
  "--sidebar",
  "--sidebar-foreground",
  "--sidebar-primary",
  "--sidebar-primary-foreground",
  "--sidebar-accent",
  "--sidebar-accent-foreground",
  "--sidebar-border",
  "--sidebar-ring",
  "--highlight",
  "--pixel-blast-color",
  "--success",
  "--warning",
  "--error",
  "--info",
  "--trend-positive",
  "--trend-negative",
  "--trend-neutral",
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
  "--scrollbar-thumb",
  "--scrollbar-thumb-hover",
  "--quick-scan-flash-dim",
  "--quick-scan-flash-bright",
  "--button-glow-strong",
  "--button-glow-soft",
  "--button-glow-subtle",
  "--splash-primary",
  "--splash-primary-muted",
  "--splash-primary-faint",
  "--splash-secondary",
  "--splash-secondary-faint",
  "--brand-discord",
  "--brand-wecom",
  "--brand-feishu",
  "--brand-github-star",
  "--support-scratch-surface",
  "--support-scratch-pattern",
  "--support-scratch-prompt",
  "--support-confetti-1",
  "--support-confetti-2",
  "--support-confetti-3",
  "--support-confetti-4",
  "--chart-tooltip-shadow",
  "--chart-tooltip-shadow-lg",
  "--auth-boot-background",
  "--auth-boot-background-muted",
  "--auth-boot-ring-muted",
  "--auth-boot-title-start",
  "--auth-boot-title-end",
  "--auth-boot-title-muted-start",
  "--auth-boot-title-muted-end",
  "--auth-boot-status",
  "--auth-boot-progress-start",
  "--auth-boot-progress-end",
  "--terminal-log-info",
  "--terminal-log-error",
  "--terminal-log-foreground",
  "--terminal-log-background",
  "--terminal-log-muted",
  "--terminal-log-highlight",
  "--terminal-log-highlight-foreground",
  "--overview-agent-globe-background",
  "--media-overlay-background",
  "--media-overlay-gradient",
  "--media-overlay-foreground",
] as const

function readThemeFile(fileName: string): string {
  return readFileSync(path.join(THEME_DIR, fileName), "utf8")
}

function extractTokenNames(css: string): string[] {
  return Array.from(css.matchAll(/^\s*(--[\w-]+)\s*:/gm), (match) => match[1])
}

function extractTokenValue(css: string, token: string): string | undefined {
  return css.match(new RegExp(`^\\s*${token}:\\s*([^;]+);`, "m"))?.[1]
}

function extractOklchChannels(css: string, token: string): { chroma: number; hue: number } {
  const value = extractTokenValue(css, token)
  const match = value?.match(/oklch\(\s*[\d.]+%?\s+([\d.]+)\s+([\d.]+)/)

  expect(match, `${token} should be an absolute oklch color`).toBeTruthy()

  return {
    chroma: Number(match?.[1]),
    hue: Number(match?.[2]),
  }
}

function hueIsBetween(hue: number, start: number, end: number): boolean {
  return start <= end ? hue >= start && hue <= end : hue >= start || hue <= end
}

describe("theme token structure", () => {
  it("loads shared theme tokens before runtime theme overrides", () => {
    const indexCss = readThemeFile("index.css")

    expect(indexCss.indexOf('@import "./base.css";')).toBeLessThan(
      indexCss.indexOf('@import "./lunafox-light.css";')
    )
  })

  it("keeps invariant tokens in the shared theme base instead of duplicating them per runtime theme", () => {
    const baseCss = readThemeFile("base.css")

    for (const token of COMMON_THEME_TOKENS) {
      expect(baseCss).toContain(`${token}:`)
    }

    for (const fileName of RUNTIME_THEME_FILES) {
      const css = readThemeFile(fileName)

      for (const token of COMMON_THEME_TOKENS) {
        expect(css).not.toMatch(new RegExp(`^\\s*${token}:`, "m"))
      }
    }
  })

  it("keeps the shared LunaFox shape baseline aligned with the Primer-like radius scale", () => {
    const baseCss = readThemeFile("base.css")

    const radiusExpectations = {
      "--radius": "6px",
      "--radius-control": "6px",
      "--radius-control-subtle": "3px",
      "--radius-surface": "6px",
      "--radius-overlay": "12px",
      "--radius-overlay-arrow": "6px",
      "--radius-badge": "3px",
    } as const

    for (const [token, value] of Object.entries(radiusExpectations)) {
      expect(
        extractTokenValue(baseCss, token),
        `${token} should use the shared Primer-like radius scale`
      ).toBe(value)
    }

    expect(extractTokenValue(baseCss, "--radius-pill")).toBe("9999px")
    expect(extractTokenValue(baseCss, "--radius-round")).toBe("9999px")
  })

  it("keeps every runtime theme token-complete after shared base extraction", () => {
    for (const fileName of RUNTIME_THEME_FILES) {
      const css = readThemeFile(fileName)
      const tokenNames = new Set(extractTokenNames(css))

      for (const token of REQUIRED_RUNTIME_TOKENS) {
        expect(tokenNames.has(token), `${fileName} should define ${token}`).toBe(true)
      }
    }
  })

  it("keeps shared interaction feedback on the approved indigo token pair", () => {
    const accentTokenExpectations = {
      "lunafox-light.css": {
        background: "oklch(0.567 0.212 276.074)",
        foreground: "oklch(0.985 0 0)",
      },
      "lunafox-dark.css": {
        background: "oklch(0.72 0.14 276.074)",
        foreground: "oklch(0.17 0.02 276.074)",
      },
    } as const

    for (const [fileName, expected] of Object.entries(accentTokenExpectations)) {
      const css = readThemeFile(fileName)

      expect(extractTokenValue(css, "--interaction-accent")).toBe(expected.background)
      expect(extractTokenValue(css, "--interaction-accent-foreground")).toBe(expected.foreground)
      expect(extractTokenValue(css, "--ring")).toBe("var(--interaction-accent)")
      expect(extractTokenValue(css, "--sidebar-primary")).toBe("var(--interaction-accent)")
      expect(extractTokenValue(css, "--sidebar-primary-foreground")).toBe(
        "var(--interaction-accent-foreground)"
      )
      expect(extractTokenValue(css, "--sidebar-ring")).toBe("var(--interaction-accent)")
      expect(extractTokenValue(css, "--highlight")).toBe("var(--interaction-accent)")
    }
  })

  it("separates trend semantics from generic status colors in every runtime theme", () => {
    for (const fileName of RUNTIME_THEME_FILES) {
      const css = readThemeFile(fileName)

      expect(extractTokenValue(css, "--trend-positive")).toContain("oklch(")
      expect(extractTokenValue(css, "--trend-negative")).toContain("oklch(")
      expect(extractTokenValue(css, "--trend-neutral")).toContain("var(--muted-foreground)")
    }
  })

  it("derives severity state variants from their owning severity base token", () => {
    for (const fileName of RUNTIME_THEME_FILES) {
      const css = readThemeFile(fileName)

      for (const level of ["critical", "high", "medium", "low", "info"]) {
        expect(extractTokenValue(css, `--severity-${level}-background`)).toContain(
          `var(--severity-${level})`
        )
        expect(extractTokenValue(css, `--severity-${level}-border`)).toContain(
          `var(--severity-${level})`
        )
        expect(extractTokenValue(css, `--severity-${level}-hover`)).toContain(
          `var(--severity-${level})`
        )
      }
    }
  })

  it("uses a named theme token for the overview agent globe shell background", () => {
    for (const fileName of RUNTIME_THEME_FILES) {
      const css = readThemeFile(fileName)

      expect(css).toContain("--overview-agent-globe-background:")
      expect(extractTokenValue(css, "--overview-agent-globe-background")).toContain("var(--card)")
      expect(css).toContain("background: var(--overview-agent-globe-background);")
    }
  })

  it("keeps splash accents out of saturated magenta-purple hues", () => {
    for (const fileName of RUNTIME_THEME_FILES) {
      const css = readThemeFile(fileName)

      for (const token of SPLASH_ACCENT_TOKENS) {
        const color = extractOklchChannels(css, token)
        const isSaturatedMagentaPurple =
          color.chroma > 0.12 && hueIsBetween(color.hue, 285, 345)

        expect(isSaturatedMagentaPurple, `${fileName} ${token} should not read as hot pink/purple`).toBe(
          false
        )
      }
    }
  })

  it("keeps decorative support confetti below persistent CTA saturation", () => {
    for (const fileName of RUNTIME_THEME_FILES) {
      const css = readThemeFile(fileName)

      for (const token of SUPPORT_CONFETTI_TOKENS) {
        const color = extractOklchChannels(css, token)

        expect(color.chroma, `${fileName} ${token} should stay celebratory without dominating UI`).toBeLessThanOrEqual(
          0.17
        )
      }
    }
  })

  it("keeps standalone product effects out of saturated magenta-purple hues", () => {
    for (const fileName of RUNTIME_THEME_FILES) {
      const css = readThemeFile(fileName)

      for (const token of PRODUCT_EFFECT_TOKENS) {
        const color = extractOklchChannels(css, token)
        const isSaturatedMagentaPurple =
          color.chroma > 0.12 && hueIsBetween(color.hue, 285, 345)

        expect(isSaturatedMagentaPurple, `${fileName} ${token} should not push the theme toward pink/purple`).toBe(
          false
        )
      }
    }
  })

})
