export const COLOR_THEMES = [
  { id: "bauhaus", name: "Bauhaus", color: "#1A1D21", colors: ["#F4F5F7", "#1A1D21", "#E1E5EA", "#9CA3AF"], isDark: false },
] as const

export type ColorThemeId = "bauhaus"

export const DEFAULT_COLOR_THEME_ID: ColorThemeId = "bauhaus"
export const COLOR_THEME_COOKIE_KEY = "color-theme"

const COLOR_THEME_IDS = new Set<string>(COLOR_THEMES.map((theme) => theme.id))

export function isColorThemeId(value: string | null | undefined): value is ColorThemeId {
  return typeof value === "string" && COLOR_THEME_IDS.has(value)
}

export function resolveColorThemeId(value: string | null | undefined): ColorThemeId {
  return isColorThemeId(value) ? value : DEFAULT_COLOR_THEME_ID
}

export function isDarkColorTheme(themeId: ColorThemeId): boolean {
  return COLOR_THEMES.find((theme) => theme.id === themeId)?.isDark ?? false
}

