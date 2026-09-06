import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/card.tsx"), "utf8")

describe("card contract", () => {
  it("keeps the card shell theme-driven instead of hard-coding square corners", () => {
    expect(source).toContain("radius-surface")
    expect(source).toContain("border-t-2")
    expect(source).toContain("shadow-2xs")
    expect(source).toContain("data-card-variant")
    expect(source).toContain("variant: {")
    expect(source).toContain("before:content-[attr(data-card-index)]")
  })

  it("keeps card title and description styling on shared typography roles", () => {
    expect(source).toContain('from "@/lib/typography"')
    expect(source).toContain("textRole.sectionTitle")
    expect(source).toContain("textRole.bodySubtle")
    expect(source).not.toContain("tracking-[-0.01em]")
    expect(source).not.toContain("tracking-[0.14em]")
    expect(source).not.toContain("uppercase")
  })

  it("adds semantic card aliases while preserving current shell styling", () => {
    expect(source).toContain("shell:")
    expect(source).toContain("metric:")
    expect(source).toContain("data-card-variant")
  })

  it("keeps the shared production panel rhythm on the default card shell", () => {
    expect(source).toContain("gap-6")
    expect(source).toContain("py-6")
    expect(source).toContain("px-6")
    expect(source).not.toContain("p-5")
  })

  it("owns compact header and footer rhythm instead of forcing important padding utilities", () => {
    expect(source).toContain("density?: CardSectionDensity")
    expect(source).toContain("data-density")
    expect(source).toContain("compact")
    expect(source).toContain("py-2.5")
  })
})
