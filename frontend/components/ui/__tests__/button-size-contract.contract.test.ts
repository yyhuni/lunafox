import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "lib/ui/button-size-contract.ts"), "utf8")

describe("button size contract", () => {
  it("owns shared structural button sizes for controls and action placeholders", () => {
    expect(source).toContain('default: "h-9"')
    expect(source).toContain('sm: "h-8"')
    expect(source).toContain('lg: "h-10"')
    expect(source).toContain('"action-card": "h-full min-h-16 w-full"')
    expect(source).toContain('"icon-sm": "size-8"')
    expect(source).toContain('icon: "size-9"')
    expect(source).toContain('"icon-lg": "size-10"')
  })

  it("classifies icon-only sizes separately from text-bearing button shells", () => {
    expect(source).toContain("buttonIconOnlySizes")
    expect(source).toContain('["chip-icon", "icon-sm", "icon", "icon-lg"]')
  })
})
