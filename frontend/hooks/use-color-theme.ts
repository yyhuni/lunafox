import { useCallback, useContext, useEffect, useMemo, useState } from "react"

import { ThemeProviderContext } from "@/components/providers/theme-provider"
import {
  COLOR_THEMES,
  DEFAULT_THEME_MODE_ID,
  THEME_MODES,
  applyColorThemeToRoot,
  createColorThemeCookie,
  createThemeModeCookie,
  deriveColorThemeSelection,
  getColorTheme,
  resolveColorThemeId,
  resolveColorThemeIdForSelection,
  resolveThemeModeId,
  type ColorThemeId,
  type ThemeModeId,
} from "@/lib/color-themes"

export { COLOR_THEMES, THEME_MODES }
export type { ColorThemeId, ThemeModeId }

function syncStandaloneTheme(themeId: ColorThemeId, mode: ThemeModeId) {
  if (typeof document === "undefined") {
    return
  }

  applyColorThemeToRoot(document.documentElement, themeId)
  document.cookie = createColorThemeCookie(themeId)
  document.cookie = createThemeModeCookie(mode)
}

function getSystemPrefersDark() {
  return typeof window !== "undefined"
    && typeof window.matchMedia === "function"
    && window.matchMedia("(prefers-color-scheme: dark)").matches
}

export function useColorTheme() {
  const context = useContext(ThemeProviderContext)
  const [fallbackMode, setFallbackMode] = useState<ThemeModeId>(DEFAULT_THEME_MODE_ID)
  const [fallbackSystemPrefersDark, setFallbackSystemPrefersDark] = useState(getSystemPrefersDark)
  const [fallbackMounted, setFallbackMounted] = useState(false)
  const fallbackTheme = resolveColorThemeIdForSelection(
    { mode: fallbackMode },
    fallbackSystemPrefersDark
  )

  useEffect(() => {
    if (context) {
      return
    }

    syncStandaloneTheme(fallbackTheme, fallbackMode)
    setFallbackMounted(true)
  }, [context, fallbackMode, fallbackTheme])

  useEffect(() => {
    if (context || typeof window === "undefined" || typeof window.matchMedia !== "function") {
      return
    }

    const media = window.matchMedia("(prefers-color-scheme: dark)")
    const handleChange = () => setFallbackSystemPrefersDark(media.matches)

    handleChange()
    media.addEventListener("change", handleChange)

    return () => media.removeEventListener("change", handleChange)
  }, [context])

  const fallbackSetTheme = useCallback((themeId: ColorThemeId) => {
    const nextSelection = deriveColorThemeSelection(resolveColorThemeId(themeId))

    setFallbackMode(nextSelection.mode)
  }, [])
  const fallbackSetMode = useCallback((mode: ThemeModeId) => {
    setFallbackMode(resolveThemeModeId(mode))
  }, [])
  const fallbackValue = useMemo(
    () => ({
      theme: fallbackTheme,
      mode: fallbackMode,
      setTheme: fallbackSetTheme,
      setMode: fallbackSetMode,
      themes: COLOR_THEMES,
      modes: THEME_MODES,
      currentTheme: getColorTheme(fallbackTheme),
      mounted: fallbackMounted,
    }),
    [
      fallbackMode,
      fallbackMounted,
      fallbackSetMode,
      fallbackSetTheme,
      fallbackTheme,
    ]
  )

  return context ?? fallbackValue
}
