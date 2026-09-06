import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/hover-card.tsx"), "utf8")

describe("hover-card contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses shared overlay style tokens", () => {
    expect(source).toContain("from \"@/lib/ui/overlay-styles\"")
    expect(source).toContain("floatingContentMotionClassName")
    expect(source).toContain("floatingSurfaceClassName")
  })

  it("uses Base UI PreviewCard as the hover card primitive backend", () => {
    expect(source).toContain('from "@base-ui/react/preview-card"')
    expect(source).not.toContain("@radix-ui/react-hover-card")
    expect(source).toContain("HoverCardPrimitive.Positioner")
    expect(source).toContain("HoverCardPrimitive.Popup")
  })

  it("uses Base UI render directly instead of Radix-style asChild compatibility", () => {
    expect(source).not.toContain("asChild")
    expect(source).not.toContain("render={asChild ? child : render}")
  })
})
