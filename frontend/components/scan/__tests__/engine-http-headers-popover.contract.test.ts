import { readFileSync } from "node:fs"
import { resolve } from "node:path"
import { describe, expect, it } from "vitest"

// Source-level contract: lock the non-obvious layout and positioning
// constraints that visual review caught, so refactors cannot silently
// regress them.
const source = readFileSync(
  resolve(__dirname, "../engine-http-headers-popover.tsx"),
  "utf8",
)

describe("engine-http-headers-popover contract", () => {
  it("keeps padding on the scroll container so input focus rings are not clipped", () => {
    // An overflow-y-auto box also clips the other axis (CSS forces the
    // visible axis to auto), so the 3px focus ring needs container padding.
    expect(source).toContain("overflow-y-auto p-1")
  })

  it("anchors the popover with fixed positioning and collision padding inside drawers", () => {
    // The control renders inside scan drawers whose animated containers
    // carry transforms; fixed positioning keeps the anchor math correct.
    expect(source).toContain('positionMethod="fixed"')
    expect(source).toContain("collisionPadding={8}")
  })

  it("keeps the shared default trigger gap instead of a local sideOffset", () => {
    // Every in-form popover (duration picker, enum multi-select) uses the
    // shared PopoverContent default sideOffset of 4; a local override made
    // this popover's gap visibly mismatch its siblings.
    expect(source).not.toContain("sideOffset=")
  })

  it("does not reintroduce Radix-style state selectors that Base UI never sets", () => {
    expect(source).not.toContain("data-[state=open]")
    expect(source).not.toContain("data-[state=closed]")
  })

  it("renders only shared overlay surface styling instead of local borders and shadows", () => {
    // floatingSurfaceClassName already provides border/background/shadow/z.
    expect(source).not.toContain("bg-popover")
    expect(source).not.toContain("shadow-lg")
  })

  it("keeps the muted-band toolbar/footer pattern shared with EngineDurationInput", () => {
    expect(source).toContain("bg-muted/30")
    expect(source).toContain("bg-muted/20")
  })

  it("keeps all popover controls on one compact 28px tier", () => {
    // The popover deliberately sits one step below the form's 32px inputs,
    // on the tier shared with EngineDurationInput via one imported constant
    // so the two popovers can never drift apart.
    expect(source).toContain('from "@/lib/ui/compact-control-tier"')
    expect((source.match(/compactControlTier/g) ?? []).length).toBeGreaterThanOrEqual(8)
    expect(source).toContain('"size-7 text-muted-foreground hover:text-destructive"')
  })
})
