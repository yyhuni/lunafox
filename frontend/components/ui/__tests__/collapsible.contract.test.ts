import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/collapsible.tsx"), "utf8")

describe("collapsible contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("data-slot=\"collapsible\"")
  })

  it("uses Base UI Collapsible as the shared primitive baseline", () => {
    expect(source).toContain('from "@base-ui/react/collapsible"')
    expect(source).toContain("CollapsiblePrimitive.Root")
    expect(source).toContain("CollapsiblePrimitive.Trigger")
    expect(source).toContain("CollapsiblePrimitive.Panel")
    expect(source).not.toContain("@radix-ui/react-collapsible")
  })

  it("hard-cuts Radix-style asChild compatibility from the shared wrapper", () => {
    expect(source).not.toContain("asChild")
    expect(source).not.toContain("render={asChild ? child : render}")
  })
})
