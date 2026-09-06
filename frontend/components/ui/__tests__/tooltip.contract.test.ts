import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/tooltip.tsx"), "utf8")
const overlaySource = readFileSync(path.resolve(process.cwd(), "lib/ui/overlay-styles.ts"), "utf8")

describe("tooltip contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses shared overlay motion tokens", () => {
    expect(source).toContain("from \"@/lib/ui/overlay-styles\"")
    expect(source).toContain("floatingContentMotionClassName")
    expect(source).toContain("tooltipSurfaceClassName")
    expect(source).toContain("tooltipArrowClassName")
  })

  it("keeps shadcn/base tooltip positioning and compact surface geometry", () => {
    expect(source).toContain('side = "top"')
    expect(source).toContain("sideOffset = 4")
    expect(source).toContain('align = "center"')
    expect(source).toContain("alignOffset = 0")
    expect(source).toContain('className="isolate z-50"')
    expect(overlaySource).toContain("inline-flex w-fit max-w-xs")
    expect(overlaySource).toContain("items-center gap-1.5")
    expect(overlaySource).toContain("has-data-[slot=kbd]:pr-1.5")
    expect(overlaySource).toContain("rounded-[2px]")
    expect(overlaySource).not.toContain("text-balance")
    expect(overlaySource).not.toContain("radius-overlay-arrow")
  })

  it("uses Base UI as the tooltip primitive backend", () => {
    expect(source).toContain('from "@base-ui/react/tooltip"')
    expect(source).not.toContain("@radix-ui/react-tooltip")
    expect(source).toContain("TooltipPrimitive.Positioner")
    expect(source).toContain("TooltipPrimitive.Popup")
  })

  it("uses Base UI delay and render props without legacy aliases", () => {
    expect(source).not.toContain("asChild")
    expect(source).not.toContain("delayDuration")
    expect(source).not.toContain("render={asChild ? child : render}")
  })

  it("uses Base UI transform origin variables in shared tooltip styles", () => {
    expect(overlaySource).toContain("origin-(--transform-origin)")
    expect(overlaySource).toContain("data-[side=top]:-bottom-2.5")
    expect(overlaySource).toContain("data-[side=right]:-left-1")
    expect(overlaySource).not.toContain("--radix-tooltip")
  })
})
