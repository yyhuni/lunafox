import React from "react"
import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { afterEach, describe, expect, it } from "vitest"

import { useColorTheme } from "@/hooks/use-color-theme"
import { COLOR_THEMES } from "@/lib/color-themes"

afterEach(() => {
  document.cookie = "color-theme=; Path=/; Max-Age=0; SameSite=Lax"
  document.cookie = "theme-mode=; Path=/; Max-Age=0; SameSite=Lax"
  document.cookie = "theme-palette=; Path=/; Max-Age=0; SameSite=Lax"
  document.documentElement.removeAttribute("data-theme")
  document.documentElement.classList.remove("dark")
})

function StandaloneThemeProbe() {
  const { mounted, theme, mode, setMode, currentTheme, themes } = useColorTheme()

  return React.createElement(
    "button",
    {
      type: "button",
      onClick: () => setMode("dark"),
      "data-mounted": mounted ? "true" : "false",
      "data-theme-id": theme,
      "data-theme-mode": mode,
      "data-theme-name": currentTheme.name,
      "data-theme-count": String(themes.length),
    },
    theme
  )
}

describe("use-color-theme contract", () => {
  it("provides a standalone fallback and applies theme changes", async () => {
    render(React.createElement(StandaloneThemeProbe))

    await waitFor(() => {
    expect(screen.getByRole("button")).toHaveAttribute("data-mounted", "true")
  })

    expect(screen.getByRole("button")).toHaveAttribute("data-theme-id", "lunafox-light")
    expect(screen.getByRole("button")).toHaveAttribute("data-theme-mode", "system")
    expect(screen.getByRole("button")).toHaveAttribute("data-theme-count", String(COLOR_THEMES.length))

    fireEvent.click(screen.getByRole("button"))

    await waitFor(() => {
    expect(screen.getByRole("button")).toHaveAttribute("data-theme-id", "lunafox-dark")
    })

    expect(screen.getByRole("button")).toHaveAttribute("data-theme-mode", "dark")
    expect(document.documentElement.getAttribute("data-theme")).toBe("lunafox-dark")
    expect(document.documentElement.classList.contains("dark")).toBe(true)
    expect(document.cookie).toContain("color-theme=lunafox-dark")
    expect(document.cookie).toContain("theme-mode=dark")
    expect(document.cookie).not.toContain("theme-palette=")
  })
})
