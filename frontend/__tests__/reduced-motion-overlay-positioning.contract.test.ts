import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const globals = readFileSync(path.resolve(process.cwd(), "app/globals.css"), "utf8")

const sharedOverlayOwners = ["dropdown-menu", "hover-card", "popover", "select", "tooltip"] as const

function positionerSlotsIn(relativePath: string): string[] {
  const source = readFileSync(path.resolve(process.cwd(), relativePath), "utf8")
  return Array.from(source.matchAll(/data-slot="([a-z-]+-positioner)"/g), (match) => match[1])
}

describe("reduced-motion overlay positioning contract", () => {
  it("keeps the universal reduced-motion reset off overlay positioners", () => {
    // Base UI writes the anchor offset into an inline transform on the positioner.
    // Resetting that transform under `prefers-reduced-motion: reduce` strands every
    // menu, popover, tooltip and select at the viewport origin, so the reset must
    // skip `-positioner` slots.
    expect(globals).toContain('html[data-theme] [data-slot]:where(:not([data-slot$="-positioner"]))')
    expect(globals).not.toMatch(/html\[data-theme\] \[data-slot\],/)

    // The tabs indicator owns a transform for its active-trigger travel. `:where()`
    // keeps the reset's specificity unchanged so the later tabs rule still wins.
    const resetIndex = globals.indexOf('html[data-theme] [data-slot]:where(')
    const tabsIndicatorIndex = globals.indexOf('html[data-theme] [data-slot="tabs-active-indicator"]')
    expect(resetIndex).toBeGreaterThan(-1)
    expect(tabsIndicatorIndex).toBeGreaterThan(resetIndex)
  })

  it("keeps every shared overlay positioner on the suffix the reset excludes", () => {
    const slots = sharedOverlayOwners.flatMap((owner) => positionerSlotsIn(`components/ui/${owner}.tsx`))

    expect(slots).toEqual(
      expect.arrayContaining([
        "dropdown-menu-positioner",
        "dropdown-menu-sub-positioner",
        "hover-card-positioner",
        "popover-positioner",
        "select-positioner",
        "tooltip-positioner",
      ]),
    )
    expect(slots.every((slot) => slot.endsWith("-positioner"))).toBe(true)
  })
})
