import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/scroll-area.tsx"), "utf8")

describe("scroll-area contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses the Base UI scroll area backend instead of Radix", () => {
    expect(source).toContain('from "@base-ui/react/scroll-area"')
    expect(source).not.toContain("@radix-ui/react-scroll-area")
    expect(source).toContain("ScrollAreaPrimitive.Content")
  })

  it("lets constrained surfaces override the Base UI content intrinsic width", () => {
    expect(source).toContain("contentClassName")
    expect(source).toContain("<ScrollAreaPrimitive.Content")
    expect(source).toContain("className={contentClassName}")
  })

  it("keeps custom scrollbars stable and perceptible", () => {
    expect(source).toContain("scrollAreaType")
    expect(source).toContain("data-scroll-type")
    expect(source).toContain("--scroll-hide-delay")
    expect(source).toContain("duration-200")
    expect(source).toContain("data-[scrolling]:opacity-100")
    expect(source).toContain("data-[hovering]:opacity-100")
    expect(source).toContain("data-[has-overflow-y]:")
    expect(source).toContain("data-[has-overflow-x]:")
    expect(source).toContain("data-[has-overflow-y]:opacity-100")
    expect(source).toContain("data-[has-overflow-x]:opacity-100")
    expect(source).toContain("h-full w-2 border-l border-l-transparent p-px")
    expect(source).toContain("h-2 flex-col border-t border-t-transparent p-px")
    expect(source).toContain("bg-border/70")
    expect(source).toContain("group-hover/scrollbar:bg-border")
  })
})
