import React from "react"
import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { ThemeProvider } from "@/components/providers/theme-provider"
import { useColorTheme } from "@/hooks/use-color-theme"
import type { ColorThemeId } from "@/lib/color-themes"

afterEach(() => {
  vi.restoreAllMocks()
  document.cookie = "color-theme=; Path=/; Max-Age=0; SameSite=Lax"
  document.cookie = "theme-mode=; Path=/; Max-Age=0; SameSite=Lax"
  document.cookie = "theme-palette=; Path=/; Max-Age=0; SameSite=Lax"
  document.documentElement.removeAttribute("data-theme")
  document.documentElement.classList.remove("dark")
})

function ThemeProbe() {
  const { mounted, theme, mode, setMode } = useColorTheme()

  return React.createElement(
    "button",
    {
      type: "button",
      onClick: () => setMode("dark"),
      "data-mounted": mounted ? "true" : "false",
      "data-theme-id": theme,
      "data-theme-mode": mode,
    },
    theme
  )
}

function PassiveThemeReadProbe() {
  const { theme, setMode } = useColorTheme()
  const [themeSeenByEffect, setThemeSeenByEffect] = React.useState<string | null>(null)

  React.useEffect(() => {
    setThemeSeenByEffect(document.documentElement.getAttribute("data-theme"))
  }, [theme])

  return React.createElement(
    "button",
    {
      type: "button",
      onClick: () => setMode("dark"),
      "data-theme-id": theme,
      "data-theme-seen-by-effect": themeSeenByEffect ?? "",
    },
    theme
  )
}

describe("theme-provider contract", () => {
  it("normalizes legacy theme state and persists theme changes", async () => {
    render(
      React.createElement(
        ThemeProvider,
        { initialTheme: "bauhaus" as ColorThemeId },
        React.createElement(ThemeProbe)
      )
    )

    await waitFor(() => {
      expect(screen.getByRole("button")).toHaveAttribute("data-mounted", "true")
    })

    expect(document.documentElement.getAttribute("data-theme")).toBe("lunafox-light")
    expect(screen.getByRole("button")).toHaveAttribute("data-theme-mode", "system")

    fireEvent.click(screen.getByRole("button"))

    await waitFor(() => {
    expect(screen.getByRole("button")).toHaveAttribute("data-theme-id", "lunafox-dark")
    })

    expect(document.documentElement.getAttribute("data-theme")).toBe("lunafox-dark")
    expect(document.documentElement.classList.contains("dark")).toBe(true)
    expect(document.cookie).toContain("color-theme=lunafox-dark")
    expect(document.cookie).toContain("theme-mode=dark")
    expect(document.cookie).not.toContain("theme-palette=")
  })

  it("syncs the document theme before passive visual effects read theme tokens", async () => {
    render(
      React.createElement(
        ThemeProvider,
        null,
        React.createElement(PassiveThemeReadProbe)
      )
    )

    await waitFor(() => {
      expect(screen.getByRole("button")).toHaveAttribute("data-theme-seen-by-effect", "lunafox-light")
    })

    fireEvent.click(screen.getByRole("button"))

    await waitFor(() => {
      expect(screen.getByRole("button")).toHaveAttribute("data-theme-id", "lunafox-dark")
    })
    expect(screen.getByRole("button")).toHaveAttribute("data-theme-seen-by-effect", "lunafox-dark")
  })

  it("falls back from removed browser theme state without server cookie props", async () => {
    document.cookie = "color-theme=rose-cocoa-dark; Path=/; SameSite=Lax"
    document.cookie = "theme-mode=dark; Path=/; SameSite=Lax"
    document.cookie = "theme-palette=rose-cocoa; Path=/; SameSite=Lax"
    document.documentElement.setAttribute("data-theme", "rose-cocoa-dark")
    document.documentElement.classList.add("dark")

    render(
      React.createElement(
        ThemeProvider,
        null,
        React.createElement(ThemeProbe)
      )
    )

    await waitFor(() => {
      expect(screen.getByRole("button")).toHaveAttribute("data-mounted", "true")
    })

    expect(screen.getByRole("button")).toHaveAttribute("data-theme-id", "lunafox-dark")
    expect(screen.getByRole("button")).toHaveAttribute("data-theme-mode", "dark")
    expect(document.documentElement.getAttribute("data-theme")).toBe("lunafox-dark")
    expect(document.documentElement.classList.contains("dark")).toBe(true)
  })

  it("does not toggle data-theme-sync when the initial document theme already matches", async () => {
    document.documentElement.setAttribute("data-theme", "lunafox-light")
    const setAttributeSpy = vi.spyOn(document.documentElement, "setAttribute")

    render(
      React.createElement(
        ThemeProvider,
        null,
        React.createElement(ThemeProbe)
      )
    )

    await waitFor(() => {
      expect(screen.getByRole("button")).toHaveAttribute("data-mounted", "true")
    })

    expect(setAttributeSpy).not.toHaveBeenCalledWith("data-theme-sync", "true")
    expect(document.cookie).toContain("color-theme=lunafox-light")
    expect(document.cookie).toContain("theme-mode=system")
    expect(document.cookie).not.toContain("theme-palette=")
  })

  it("tracks system preference changes without changing the selected mode", async () => {
    let prefersDark = false
    let changeHandler: (() => void) | undefined
    const mediaQuery = {
      matches: prefersDark,
      media: "(prefers-color-scheme: dark)",
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn((_event: string, handler: () => void) => {
        changeHandler = handler
      }),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    } as unknown as MediaQueryList

    vi.spyOn(window, "matchMedia").mockReturnValue(mediaQuery)

    render(
      React.createElement(
        ThemeProvider,
        null,
        React.createElement(ThemeProbe)
      )
    )

    await waitFor(() => {
      expect(screen.getByRole("button")).toHaveAttribute("data-theme-mode", "system")
      expect(screen.getByRole("button")).toHaveAttribute("data-theme-id", "lunafox-light")
    })

    prefersDark = true
    Object.defineProperty(mediaQuery, "matches", { configurable: true, value: prefersDark })
    changeHandler?.()

    await waitFor(() => {
      expect(screen.getByRole("button")).toHaveAttribute("data-theme-mode", "system")
      expect(screen.getByRole("button")).toHaveAttribute("data-theme-id", "lunafox-dark")
    })
    expect(document.cookie).toContain("theme-mode=system")
  })
})
