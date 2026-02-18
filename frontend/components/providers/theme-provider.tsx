"use client"

import type React from "react"

type ThemeProviderProps = {
  children: React.ReactNode
}

/**
 * Theme provider placeholder
 * Bauhaus is the only theme, so no runtime theme switching is needed.
 */
export function ThemeProvider({ children }: ThemeProviderProps) {
  return <>{children}</>
}
