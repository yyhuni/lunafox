import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/color-theme-init.tsx"), "utf8")

describe("color-theme-init contract", () => {
  it("restores theme from cookie-aware init script markers", () => {
    expect(source).toContain("export function ColorThemeInit")
    expect(source).not.toContain("from \"next/script\"")
    expect(source).toContain('id="color-theme-init"')
    expect(source).toContain("dangerouslySetInnerHTML")
    expect(source).toContain("COLOR_THEME_COOKIE_KEY")
    expect(source).toContain("THEME_MODE_COOKIE_KEY")
    expect(source).toContain("COLOR_THEMES.filter((theme) => theme.isDark)")
    expect(source).toContain("window.matchMedia")
    expect(source).toContain("legacyDarkThemes")
    expect(source).toContain("legacyLightThemes")
    expect(source).toContain('"blue-dark"')
    expect(source).toContain('"smoked-amethyst-dark"')
    expect(source).toContain('"blue-light"')
    expect(source).toContain('"smoked-amethyst-light"')
    expect(source).toContain('t==="bauhaus"||t===("shad"+"cn-default")')
    expect(source).toContain('m!=="light"&&m!=="dark"&&m!=="system"')
    expect(source).not.toContain("THEME_PALETTE")
    expect(source).not.toContain("THEME_PALETTES")
    expect(source).not.toContain("paletteThemePairs")
    expect(source).toContain("].join(\"\")")
    expect(source).not.toContain("validThemes")
    expect(source).not.toContain("validModes")
    expect(source).not.toContain("validPalettes")
    expect(source).toContain('r.setAttribute("data-theme",n)')
    expect(source).toContain('r.classList.add')
    expect(source).toContain('r.classList.remove')
  })
})
