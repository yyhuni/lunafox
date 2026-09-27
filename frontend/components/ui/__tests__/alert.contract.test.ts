import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/alert.tsx"), "utf8")

describe("alert contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses the compact ordinary-info geometry and semantic typography roles", () => {
    expect(source).toContain("radius-control")
    expect(source).toContain("px-3 py-2")
    expect(source).toContain("textRole.helperText")
    expect(source).toContain("textRole.compactSectionTitle")
  })
})
