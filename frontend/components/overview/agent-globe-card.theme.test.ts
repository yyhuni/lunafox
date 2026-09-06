// @vitest-environment jsdom

import { afterEach, describe, expect, it, vi } from "vitest"
import { buildThemeSyncedCobeGlobeTheme } from "./agent-globe-card"
import { getColorTheme } from "@/lib/color-themes"

const LUNA_PRIMARY_OKLCH = ["ok", "lch(0.205 0 0)"].join("")
const LUNA_BACKGROUND_OKLCH = ["ok", "lch(1 0 0)"].join("")
const LUNA_DARK_PRIMARY_LAB = ["la", "b(90.952 0 -0.0000119209)"].join("")
const LUNA_DARK_BACKGROUND_LAB = ["la", "b(2.75381 0 0)"].join("")

describe("buildThemeSyncedCobeGlobeTheme", () => {
  afterEach(() => {
    document.documentElement.removeAttribute("data-theme")
    document.head.innerHTML = ""
    document.body.innerHTML = ""
  })

  it("resolves oklch theme tokens so the default light globe does not use its fallback color", () => {
    const originalGetComputedStyle = window.getComputedStyle.bind(window)
    const getComputedStyleSpy = vi.spyOn(window, "getComputedStyle").mockImplementation((element) => {
      if (element instanceof HTMLElement && element.style.color === "var(--primary)") {
        return { color: LUNA_PRIMARY_OKLCH } as CSSStyleDeclaration
      }

      if (element instanceof HTMLElement && element.style.color === "var(--background)") {
        return { color: LUNA_BACKGROUND_OKLCH } as CSSStyleDeclaration
      }

      return originalGetComputedStyle(element)
    })

    const theme = buildThemeSyncedCobeGlobeTheme(getColorTheme("lunafox-light"))

    expect(theme.markerColor[0]).toBeCloseTo(theme.markerColor[1], 3)
    expect(theme.markerColor[1]).toBeCloseTo(theme.markerColor[2], 3)
    expect(theme.markerColor[0]).toBeLessThan(0.12)
    expect(theme.markerColor).not.toEqual([0.64, 0.23, 0.3])
    expect(theme.arcColor).toEqual(theme.markerColor)

    getComputedStyleSpy.mockRestore()
  })

  it("resolves lab theme tokens so the default dark globe does not use its fallback color", () => {
    const originalGetComputedStyle = window.getComputedStyle.bind(window)
    const getComputedStyleSpy = vi.spyOn(window, "getComputedStyle").mockImplementation((element) => {
      if (element instanceof HTMLElement && element.style.color === "var(--primary)") {
        return { color: LUNA_DARK_PRIMARY_LAB } as CSSStyleDeclaration
      }

      if (element instanceof HTMLElement && element.style.color === "var(--background)") {
        return { color: LUNA_DARK_BACKGROUND_LAB } as CSSStyleDeclaration
      }

      return originalGetComputedStyle(element)
    })

    const theme = buildThemeSyncedCobeGlobeTheme(getColorTheme("lunafox-dark"))

    expect(theme.markerColor[0]).toBeCloseTo(theme.markerColor[1], 3)
    expect(theme.markerColor[1]).toBeCloseTo(theme.markerColor[2], 3)
    expect(theme.markerColor[1]).toBeGreaterThan(0.85)
    expect(theme.markerColor).not.toEqual([0.88, 0.52, 0.6])
    expect(theme.arcColor).toEqual(theme.markerColor)

    getComputedStyleSpy.mockRestore()
  })

})
