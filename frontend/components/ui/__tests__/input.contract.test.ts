import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/input.tsx"), "utf8")

describe("input contract", () => {
  it("keeps input radius under theme control in the component", () => {
    expect(source).toContain("radius-control")
  })

  it("adds semantic input variants and exposes the resolved input surface", () => {
    expect(source).toContain("subtle:")
    expect(source).toContain("data-input-variant")
  })

  it("owns standard and compact input heights through shared sizes", () => {
    expect(source).toContain("size:")
    expect(source).toContain('default: "h-9"')
    expect(source).toContain('sm: "h-8"')
    expect(source).toContain("data-input-size")
  })

  it("suppresses native webkit search clear affordances in shared inputs", () => {
    expect(source).toContain("[&::-webkit-search-cancel-button]:appearance-none")
    expect(source).toContain("[&::-webkit-search-decoration]:appearance-none")
    expect(source).toContain("[&::-webkit-search-results-button]:appearance-none")
    expect(source).toContain("[&::-webkit-search-results-decoration]:appearance-none")
  })
})
