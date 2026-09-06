import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/field.tsx"), "utf8")

describe("field contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses Base UI checked-state attributes for selected field shells", () => {
    expect(source).toContain("has-data-[checked]:bg-primary/5")
    expect(source).toContain("has-data-[checked]:border-primary")
    expect(source).not.toContain("has-data-[state=checked]")
  })
})
