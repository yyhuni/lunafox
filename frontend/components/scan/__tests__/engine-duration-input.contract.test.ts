import { readFileSync } from "node:fs"
import { resolve } from "node:path"
import { describe, expect, it } from "vitest"

// Source-level contract: keep the duration popover on the shared overlay
// surface and the muted-band layout shared with the headers popover, so
// neither control can drift back to local shadow/border styling.
const source = readFileSync(
  resolve(__dirname, "../engine-duration-input.tsx"),
  "utf8",
)

describe("engine-duration-input popover contract", () => {
  it("does not layer a local shadow over the shared floating surface", () => {
    // floatingSurfaceClassName already provides border/background/shadow/z;
    // a local shadow-lg would double the elevation and drift from the
    // foundation overlay styles.
    expect(source).not.toContain("shadow-lg")
    expect(source).not.toContain("shadow-md")
  })

  it("keeps the muted-band header/footer pattern", () => {
    expect(source).toContain("bg-muted/30")
    expect(source).toContain("bg-muted/20")
  })

  it("keeps footer actions on the shared xs button size", () => {
    expect(source.match(/size="xs"/g)).toHaveLength(3)
    expect(source).not.toContain("compactControlTier")
    expect(source).toContain("px-3 py-1.5")
    expect(source).not.toContain('className="text-xs"')
  })

  it("does not reintroduce Radix-style state selectors that Base UI never sets", () => {
    expect(source).not.toContain("data-[state=open]")
    expect(source).not.toContain("data-[state=closed]")
  })
})
