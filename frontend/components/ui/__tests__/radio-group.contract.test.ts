import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/radio-group.tsx"), "utf8")

describe("radio-group contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
    expect(source).toContain("radius-round")
  })

  it("uses Base UI Radio Group and Radio as the shared primitive baseline", () => {
    expect(source).toContain('from "@base-ui/react/radio-group"')
    expect(source).toContain('from "@base-ui/react/radio"')
    expect(source).toContain("RadioGroupPrimitive")
    expect(source).toContain("RadioPrimitive.Root")
    expect(source).toContain("RadioPrimitive.Indicator")
    expect(source).toContain("data-[unchecked]:hidden")
    expect(source).not.toContain("@radix-ui/react-radio-group")
  })

  it("keeps the radio item indicator centered in a fixed control box", () => {
    expect(source).toContain("inline-flex")
    expect(source).toContain("items-center")
    expect(source).toContain("justify-center")
    expect(source).toContain("relative")
    expect(source).toContain("size-4")
    expect(source).toContain('className="absolute inset-0 flex items-center justify-center data-[unchecked]:hidden"')
    expect(source).not.toContain("left-1/2")
    expect(source).not.toContain("top-1/2")
    expect(source).not.toContain("-translate-x-1/2")
    expect(source).not.toContain("-translate-y-1/2")
  })
})
