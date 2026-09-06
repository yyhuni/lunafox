import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/common/smart-filter-input-state.ts"), "utf8")

describe("smart-filter-input-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useSmartFilterInputState")
    expect(source).toContain("from \"react\"")
  })

  it("does not rely on Radix popper wrapper markers when preserving focus", () => {
    expect(source).not.toContain("data-radix-popper-content-wrapper")
  })
})
