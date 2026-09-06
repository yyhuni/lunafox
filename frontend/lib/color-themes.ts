export const COLOR_THEMES = [
  {
    id: "lunafox-light",
    name: "LunaFox Light",
    color: "var(--primary)",
    colors: ["var(--background)", "var(--primary)", "var(--muted-foreground)", "var(--border)"],
    isDark: false,
    description: "Default light workspace with neutral surfaces, graphite structure, and signal-led accents for a calmer data cockpit.",
  },
  {
    id: "lunafox-dark",
    name: "LunaFox Dark",
    color: "var(--primary)",
    colors: ["var(--background)", "var(--primary)", "var(--muted-foreground)", "var(--border)"],
    isDark: true,
    description: "Neutral dark baseline with cooler signal accents and brighter LunaFox highlights for a focused data cockpit.",
  },
] as const

export type ColorTheme = (typeof COLOR_THEMES)[number]
export type ColorThemeId = ColorTheme["id"]

export const DEFAULT_LIGHT_COLOR_THEME_ID: ColorThemeId = "lunafox-light"
export const DEFAULT_DARK_COLOR_THEME_ID: ColorThemeId = "lunafox-dark"
export const DEFAULT_COLOR_THEME_ID: ColorThemeId = DEFAULT_LIGHT_COLOR_THEME_ID

export const THEME_MODES = [
  { id: "system", name: "System" },
  { id: "light", name: "Light" },
  { id: "dark", name: "Dark" },
] as const

export type ThemeModeId = (typeof THEME_MODES)[number]["id"]
export type ResolvedThemeModeId = Exclude<ThemeModeId, "system">

export const DEFAULT_THEME_MODE_ID: ThemeModeId = "system"
export const COLOR_THEME_COOKIE_KEY = "color-theme"
export const THEME_MODE_COOKIE_KEY = "theme-mode"
export const COLOR_THEME_DATA_ATTRIBUTE = "data-theme"
export const COLOR_THEME_COOKIE_MAX_AGE = 60 * 60 * 24 * 365
const LEGACY_DEFAULT_THEME_STORAGE_VALUE = "shad" + "cn-default"
const LEGACY_THEME_ID_MAP = new Map<string, ColorThemeId>([
  ["bauhaus", DEFAULT_COLOR_THEME_ID],
  [LEGACY_DEFAULT_THEME_STORAGE_VALUE, DEFAULT_COLOR_THEME_ID],
  ["cherry-cocoa-light", DEFAULT_LIGHT_COLOR_THEME_ID],
  ["cherry-cocoa-dark", DEFAULT_DARK_COLOR_THEME_ID],
  ["rose-cocoa-light", DEFAULT_LIGHT_COLOR_THEME_ID],
  ["rose-cocoa-dark", DEFAULT_DARK_COLOR_THEME_ID],
  ["graphite-contrast-light", DEFAULT_LIGHT_COLOR_THEME_ID],
  ["graphite-contrast-dark", DEFAULT_DARK_COLOR_THEME_ID],
  ["mist-teal-light", DEFAULT_LIGHT_COLOR_THEME_ID],
  ["mist-teal-dark", DEFAULT_DARK_COLOR_THEME_ID],
  ["ink-navy-light", DEFAULT_LIGHT_COLOR_THEME_ID],
  ["ink-navy-dark", DEFAULT_DARK_COLOR_THEME_ID],
  ["paper-ink-light", DEFAULT_LIGHT_COLOR_THEME_ID],
  ["paper-ink-dark", DEFAULT_DARK_COLOR_THEME_ID],
  ["blue-light", DEFAULT_LIGHT_COLOR_THEME_ID],
  ["blue-dark", DEFAULT_DARK_COLOR_THEME_ID],
  ["smoked-amethyst-light", DEFAULT_LIGHT_COLOR_THEME_ID],
  ["smoked-amethyst-dark", DEFAULT_DARK_COLOR_THEME_ID],
])

const COLOR_THEME_MAP = new Map<ColorThemeId, ColorTheme>(
  COLOR_THEMES.map((theme) => [theme.id, theme])
)
const COLOR_THEME_IDS = new Set<string>(COLOR_THEMES.map((theme) => theme.id))
const THEME_MODE_IDS = new Set<string>(THEME_MODES.map((mode) => mode.id))
export type ColorThemeSelection = {
  mode: ThemeModeId
}

export function isColorThemeId(value: string | null | undefined): value is ColorThemeId {
  return typeof value === "string" && COLOR_THEME_IDS.has(value)
}

export function isThemeModeId(value: string | null | undefined): value is ThemeModeId {
  return typeof value === "string" && THEME_MODE_IDS.has(value)
}

export function resolveColorThemeId(value: string | null | undefined): ColorThemeId {
  return isColorThemeId(value) ? value : DEFAULT_COLOR_THEME_ID
}

export function resolveThemeModeId(value: string | null | undefined): ThemeModeId {
  return isThemeModeId(value) ? value : DEFAULT_THEME_MODE_ID
}

export function resolveStoredColorThemeId(value: string | null | undefined): ColorThemeId {
  if (typeof value === "string") {
    const legacyThemeId = LEGACY_THEME_ID_MAP.get(value)
    if (legacyThemeId) {
      return legacyThemeId
    }
  }

  return resolveColorThemeId(value)
}

export function resolveEffectiveThemeMode(
  mode: ThemeModeId,
  systemPrefersDark: boolean
): ResolvedThemeModeId {
  if (mode === "system") {
    return systemPrefersDark ? "dark" : "light"
  }

  return mode
}

export function resolveColorThemeIdForSelection(
  selection: ColorThemeSelection,
  systemPrefersDark: boolean
): ColorThemeId {
  const effectiveMode = resolveEffectiveThemeMode(resolveThemeModeId(selection.mode), systemPrefersDark)

  return effectiveMode === "dark" ? DEFAULT_DARK_COLOR_THEME_ID : DEFAULT_LIGHT_COLOR_THEME_ID
}

export function deriveColorThemeSelection(themeId: ColorThemeId): ColorThemeSelection {
  return {
    mode: isDarkColorTheme(themeId) ? "dark" : "light",
  }
}

export function resolveStoredColorThemeSelection({
  storedMode,
  storedTheme,
}: {
  storedMode?: string | null
  storedPalette?: string | null
  storedTheme?: string | null
}): ColorThemeSelection {
  const resolvedMode = resolveThemeModeId(storedMode)
  if (isThemeModeId(storedMode)) {
    return { mode: resolvedMode }
  }

  if (!storedTheme) {
    return { mode: DEFAULT_THEME_MODE_ID }
  }

  return deriveColorThemeSelection(resolveStoredColorThemeId(storedTheme))
}

export function getColorTheme(themeId: ColorThemeId): ColorTheme {
  return COLOR_THEME_MAP.get(themeId) ?? COLOR_THEME_MAP.get(DEFAULT_COLOR_THEME_ID)!
}

export function resolveColorTheme(value: string | null | undefined): ColorTheme {
  return getColorTheme(resolveColorThemeId(value))
}

export function isDarkColorTheme(themeId: ColorThemeId): boolean {
  return getColorTheme(themeId).isDark
}

export function getDefaultThemeToggleTarget(themeId: ColorThemeId): ColorThemeId {
  return isDarkColorTheme(themeId) ? DEFAULT_LIGHT_COLOR_THEME_ID : DEFAULT_DARK_COLOR_THEME_ID
}

export function applyColorThemeToRoot(root: HTMLElement, themeId: ColorThemeId) {
  root.setAttribute(COLOR_THEME_DATA_ATTRIBUTE, themeId)
  root.classList.toggle("dark", isDarkColorTheme(themeId))
}

export function getColorThemeCookieValue(cookieSource: string | null | undefined): string | null {
  return getCookieValue(cookieSource, COLOR_THEME_COOKIE_KEY)
}

export function getThemeModeCookieValue(cookieSource: string | null | undefined): string | null {
  return getCookieValue(cookieSource, THEME_MODE_COOKIE_KEY)
}

function getCookieValue(cookieSource: string | null | undefined, key: string): string | null {
  if (!cookieSource) {
    return null
  }

  const prefix = `${key}=`
  for (const entry of cookieSource.split(";")) {
    const trimmedEntry = entry.trim()
    if (trimmedEntry.startsWith(prefix)) {
      return decodeURIComponent(trimmedEntry.slice(prefix.length))
    }
  }

  return null
}

export function createColorThemeCookie(themeId: ColorThemeId) {
  return createThemeCookie(COLOR_THEME_COOKIE_KEY, themeId)
}

export function createThemeModeCookie(mode: ThemeModeId) {
  return createThemeCookie(THEME_MODE_COOKIE_KEY, mode)
}

function createThemeCookie(key: string, value: string) {
  return `${key}=${encodeURIComponent(value)}; Path=/; Max-Age=${COLOR_THEME_COOKIE_MAX_AGE}; SameSite=Lax`
}
