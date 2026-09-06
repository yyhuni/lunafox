import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/common/smart-filter-input-menu.tsx"), "utf8")

describe("smart-filter-input-menu contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function SmartFilterInputMenu")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })
})
