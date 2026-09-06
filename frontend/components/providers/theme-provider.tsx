"use client"

import * as React from "react"

import {
  COLOR_THEMES,
  COLOR_THEME_DATA_ATTRIBUTE,
  DEFAULT_THEME_MODE_ID,
  THEME_MODES,
  applyColorThemeToRoot,
  createColorThemeCookie,
  createThemeModeCookie,
  deriveColorThemeSelection,
  getColorThemeCookieValue,
  getColorTheme,
  getThemeModeCookieValue,
  resolveColorThemeId,
  resolveColorThemeIdForSelection,
  resolveStoredColorThemeSelection,
  resolveThemeModeId,
  type ColorTheme,
  type ColorThemeId,
  type ThemeModeId,
} from "@/lib/color-themes"

type ThemeProviderProps = {
  children?: React.ReactNode
  initialTheme?: ColorThemeId
  initialMode?: ThemeModeId
}

export type ThemeProviderValue = {
  theme: ColorThemeId
  mode: ThemeModeId
  currentTheme: ColorTheme
  themes: typeof COLOR_THEMES
  modes: typeof THEME_MODES
  mounted: boolean
  setTheme: (themeId: ColorThemeId) => void
  setMode: (mode: ThemeModeId) => void
}

export const ThemeProviderContext = React.createContext<ThemeProviderValue | null>(null)

let themeSyncFrame: number | null = null

function getSystemPrefersDark() {
  return typeof window !== "undefined"
    && typeof window.matchMedia === "function"
    && window.matchMedia("(prefers-color-scheme: dark)").matches
}

function applyThemeWithoutTransitions(root: HTMLElement, themeId: ColorThemeId) {
  root.setAttribute("data-theme-sync", "true")
  applyColorThemeToRoot(root, themeId)

  if (typeof window === "undefined" || typeof window.requestAnimationFrame !== "function") {
    root.removeAttribute("data-theme-sync")
    return
  }

  if (themeSyncFrame !== null) {
    window.cancelAnimationFrame(themeSyncFrame)
  }

  themeSyncFrame = window.requestAnimationFrame(() => {
    themeSyncFrame = window.requestAnimationFrame(() => {
      root.removeAttribute("data-theme-sync")
      themeSyncFrame = null
    })
  })
}

function shouldApplyThemeToRoot(root: HTMLElement, themeId: ColorThemeId) {
  const currentThemeId = root.getAttribute(COLOR_THEME_DATA_ATTRIBUTE)
  const shouldBeDark = getColorTheme(themeId).isDark
  const hasDarkClass = root.classList.contains("dark")

  return currentThemeId !== themeId || hasDarkClass !== shouldBeDark
}

function syncDocumentTheme(themeId: ColorThemeId, mode: ThemeModeId) {
  if (typeof document === "undefined") {
    return
  }

  const root = document.documentElement

  if (shouldApplyThemeToRoot(root, themeId)) {
    applyThemeWithoutTransitions(root, themeId)
  }

  document.cookie = createColorThemeCookie(themeId)
  document.cookie = createThemeModeCookie(mode)
}

function readInitialThemeSelection({
  initialTheme,
  initialMode,
}: {
  initialTheme?: ColorThemeId
  initialMode?: ThemeModeId
}) {
  if (typeof document === "undefined") {
    return resolveStoredColorThemeSelection({
      storedMode: initialMode,
      storedTheme: initialTheme,
    })
  }

  const cookieSource = document.cookie
  const rootTheme = document.documentElement.getAttribute(COLOR_THEME_DATA_ATTRIBUTE)

  return resolveStoredColorThemeSelection({
    storedMode: getThemeModeCookieValue(cookieSource) ?? initialMode,
    storedTheme: getColorThemeCookieValue(cookieSource) ?? rootTheme ?? initialTheme,
  })
}

export function ThemeProvider({
  children,
  initialTheme,
  initialMode = DEFAULT_THEME_MODE_ID,
}: ThemeProviderProps) {
  const [selection, setSelection] = React.useState(() => readInitialThemeSelection({
    initialMode,
    initialTheme,
  }))
  const [systemPrefersDark, setSystemPrefersDark] = React.useState(getSystemPrefersDark)
  const [mounted, setMounted] = React.useState(false)
  const theme = React.useMemo(
    () => resolveColorThemeIdForSelection(selection, systemPrefersDark),
    [selection, systemPrefersDark]
  )

  React.useEffect(() => {
    setSelection(readInitialThemeSelection({
      initialMode,
      initialTheme,
    }))
  }, [initialMode, initialTheme])

  React.useLayoutEffect(() => {
    syncDocumentTheme(theme, selection.mode)
  }, [selection.mode, theme])

  React.useEffect(() => {
    setMounted(true)
  }, [])

  React.useEffect(() => {
    if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
      return
    }

    const media = window.matchMedia("(prefers-color-scheme: dark)")
    const handleChange = () => setSystemPrefersDark(media.matches)

    handleChange()
    media.addEventListener("change", handleChange)

    return () => media.removeEventListener("change", handleChange)
  }, [])

  const setTheme = React.useCallback((themeId: ColorThemeId) => {
    const resolvedTheme = resolveColorThemeId(themeId)
    const nextSelection = deriveColorThemeSelection(resolvedTheme)

    setSelection(nextSelection)
    syncDocumentTheme(resolvedTheme, nextSelection.mode)
  }, [])

  const setMode = React.useCallback((mode: ThemeModeId) => {
    setSelection((currentSelection) => ({
      ...currentSelection,
      mode: resolveThemeModeId(mode),
    }))
  }, [])

  const value = React.useMemo<ThemeProviderValue>(
    () => ({
      theme,
      mode: selection.mode,
      currentTheme: getColorTheme(theme),
      themes: COLOR_THEMES,
      modes: THEME_MODES,
      mounted,
      setTheme,
      setMode,
    }),
    [mounted, selection.mode, setMode, setTheme, theme]
  )

  return (
    <ThemeProviderContext.Provider value={value}>
      {children}
    </ThemeProviderContext.Provider>
  )
}
