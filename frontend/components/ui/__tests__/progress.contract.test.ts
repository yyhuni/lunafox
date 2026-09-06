import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/progress.tsx"), "utf8")

describe("progress contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
    expect(source).toContain("radius-pill")
  })

  it("uses Base UI Progress as the shared primitive baseline", () => {
    expect(source).toContain('from "@base-ui/react/progress"')
    expect(source).toContain("ProgressPrimitive.Root")
    expect(source).toContain("ProgressPrimitive.Indicator")
    expect(source).not.toContain("@radix-ui/react-progress")
  })

  it("uses transform as the sole dynamic percentage calculation", () => {
    expect(source).toContain("origin-left")
    expect(source).toContain("transition-[transform,background-color]")
    expect(source).toContain('style={{ width: "100%", transform: `scaleX(${progressValue / 100})` }}')
    expect(source).toContain("scaleX")
    expect(source).not.toContain("transition-all")
  })
})
