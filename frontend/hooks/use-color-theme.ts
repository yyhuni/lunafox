/**
 * Color theme hook
 * Only keep the Bauhaus single theme
 */
import { useCallback, useEffect, useState } from "react"
import { COLOR_THEMES, DEFAULT_COLOR_THEME_ID, type ColorThemeId } from "@/lib/color-themes"

export { COLOR_THEMES }
export type { ColorThemeId }

const ACTIVE_THEME: ColorThemeId = DEFAULT_COLOR_THEME_ID
const CURRENT_THEME = COLOR_THEMES[0]

function applyThemeAttribute() {
  if (typeof document === "undefined") return
  const root = document.documentElement
  root.setAttribute("data-theme", ACTIVE_THEME)
  root.classList.remove("dark")
}

export function useColorTheme() {
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    applyThemeAttribute()
    setMounted(true)
  }, [])

  const setTheme = useCallback(() => {
    applyThemeAttribute()
  }, [])

  return {
    theme: ACTIVE_THEME,
    setTheme,
    themes: COLOR_THEMES,
    currentTheme: CURRENT_THEME,
    mounted,
  }
}
