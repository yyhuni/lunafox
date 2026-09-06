import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/number-stepper-input.tsx"), "utf8")

describe("number-stepper-input contract", () => {
  it("owns a compact shared integer stepper geometry", () => {
    expect(source).toContain("export function NumberStepperInput")
    expect(source).toContain('data-slot="number-stepper-input"')
    expect(source).toContain("h-8")
    expect(source).toContain("radius-control")
    expect(source).toContain("border-input")
  })

  it("uses project icons inside embedded plus and minus actions", () => {
    expect(source).toContain('from "@/components/icons"')
    expect(source).not.toContain('from "@/components/ui/button"')
    expect(source).toContain("Minus")
    expect(source).toContain("Plus")
    expect(source).toContain("aspect-square")
    expect(source).toContain("h-[inherit]")
    expect(source).toContain('aria-label="减少"')
    expect(source).toContain('aria-label="增加"')
  })

  it("keeps embedded actions on the same input surface in dark mode", () => {
    expect(source).toContain("bg-background dark:bg-input/30")
    expect(source).toContain("border-input bg-transparent text-muted-foreground hover:bg-muted/60 hover:text-foreground")
    expect(source).not.toContain("border-input bg-background text-muted-foreground hover:bg-muted hover:text-foreground")
  })

  it("hides the browser-native number spinner", () => {
    expect(source).toContain("[appearance:textfield]")
    expect(source).toContain("[&::-webkit-inner-spin-button]:appearance-none")
    expect(source).toContain("[&::-webkit-outer-spin-button]:appearance-none")
  })

  it("clamps increment and decrement to min and max values", () => {
    expect(source).toContain("clampNumber")
    expect(source).toContain("Math.max")
    expect(source).toContain("Math.min")
  })
})
