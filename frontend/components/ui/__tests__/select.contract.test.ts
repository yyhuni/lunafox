import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/select.tsx"), "utf8")

describe("select contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses Base UI Select as the shared primitive baseline", () => {
    expect(source).toContain('from "@base-ui/react/select"')
    expect(source).not.toContain('from "@radix-ui/react-select"')
    expect(source).toContain("SelectPrimitive.Root")
    expect(source).toContain("modal = false")
    expect(source).toContain("SelectPrimitive.Positioner")
    expect(source).toContain("SelectPrimitive.Popup")
    expect(source).toContain("SelectPrimitive.List")
  })

  it("owns a shared content-fit width mode for compact selects", () => {
    expect(source).toContain("radius-control")
    expect(source).toContain("radius-overlay")
    expect(source).toContain('"radius-control data-[highlighted]:bg-accent')
    expect(source).not.toContain('"radius-control-subtle data-[highlighted]:bg-accent')
    expect(source).toContain('width = "default"')
    expect(source).toContain('width?: "default" | "content-fit"')
    expect(source).toContain('width === "default" && "min-w-[max(8rem,var(--anchor-width))]"')
    expect(source).toContain('width === "content-fit" && "w-max min-w-[var(--anchor-width)]"')
    expect(source).toContain("--anchor-width")
    expect(source).toContain("--available-height")
    expect(source).not.toContain("--radix-select-trigger-width")
    expect(source).not.toContain("--radix-select-trigger-height")
  })

  it("uses the current item-aligned popup style by default while keeping popper opt-in", () => {
    expect(source).toContain('position = "item-aligned"')
    expect(source).toContain('position?: "popper" | "item-aligned"')
    expect(source).toContain("const shouldAlignItemWithTrigger = position === \"item-aligned\" || alignItemWithTrigger")
    expect(source).toContain('alignItemWithTrigger={shouldAlignItemWithTrigger}')
    expect(source).toContain('sideOffset={shouldAlignItemWithTrigger ? 0 : sideOffset}')
  })

  it("keeps item-aligned popups in the overlay event layer without entering transformed content", () => {
    expect(source).toContain("contentContainer: HTMLElement | null")
    expect(source).toContain("itemAlignedContainer: HTMLElement | null")
    expect(source).toContain("[data-slot='drawer-content'], [data-slot='dialog-content'], [data-slot='sheet-content']")
    expect(source).toContain("[data-slot='drawer-viewport'], [data-slot='dialog-portal'], [data-slot='sheet-portal']")
    expect(source).toContain("const contentPortalContainer = shouldAlignItemWithTrigger")
    expect(source).toContain("? portalContainer?.itemAlignedContainer ?? undefined")
    expect(source).toContain(": portalContainer?.contentContainer ?? undefined")
    expect(source).toContain("container={contentPortalContainer}")
    expect(source).not.toContain("container={portalContainer?.container ?? undefined}")
  })

  it("puts the stacking layer on the positioned wrapper", () => {
    expect(source).toContain('data-slot="select-positioner"')
    expect(source).toContain('className="z-50"')
  })

  it("falls back to the document portal when no overlay owner is available", () => {
    expect(source).toContain("container={contentPortalContainer}")
    expect(source).not.toContain("container={portalContainer?.container}>")
  })
})
