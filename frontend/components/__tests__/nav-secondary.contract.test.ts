import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/nav-secondary.tsx"), "utf8")

describe("nav-secondary contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function NavSecondary")
    expect(source).toContain("from \"react\"")
  })
})
