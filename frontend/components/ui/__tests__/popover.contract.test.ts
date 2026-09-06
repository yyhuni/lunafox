import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/popover.tsx"), "utf8")

describe("popover contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses shared overlay style tokens", () => {
    expect(source).toContain("from \"@/lib/ui/overlay-styles\"")
    expect(source).toContain("floatingContentMotionClassName")
    expect(source).toContain("floatingSurfaceClassName")
  })

  it("uses Base UI as the popover primitive backend", () => {
    expect(source).toContain('from "@base-ui/react/popover"')
    expect(source).not.toContain("@radix-ui/react-popover")
    expect(source).toContain("PopoverPrimitive.Positioner")
    expect(source).toContain("PopoverPrimitive.Popup")
  })

  it("keeps the positioned floating layer above following drawer content", () => {
    expect(source).toContain('data-slot="popover-positioner"')
    expect(source).toContain('className="z-50"')
  })

  it("uses Base UI render and focus props without legacy compatibility aliases", () => {
    expect(source).not.toContain("asChild")
    expect(source).not.toContain("onOpenAutoFocus")
    expect(source).not.toContain("onCloseAutoFocus")
    expect(source).not.toContain("render={asChild ? child : render}")
    expect(source).toContain("PopoverAnchorContext")
    expect(source).toContain("anchor={anchor ?? undefined}")
  })

  it("uses Base UI geometry variables instead of Radix popover variables", () => {
    expect(source).toContain("--anchor-width")
    expect(source).not.toContain("--radix-popover")
  })

  it("lets shared popover owners opt out of trigger-width minimums without important overrides", () => {
    expect(source).toContain('minWidth?: "anchor" | "content"')
    expect(source).toContain('minWidth = "anchor"')
    expect(source).toContain('minWidth === "anchor" && "min-w-[var(--anchor-width)]"')
  })
})
