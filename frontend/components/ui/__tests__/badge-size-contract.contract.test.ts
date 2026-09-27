import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "lib/ui/badge-size-contract.ts"), "utf8")
const globals = readFileSync(path.resolve(process.cwd(), "app/globals.css"), "utf8")

describe("badge size contract", () => {
  it("keeps ordinary tags at 26px and names smaller semantic tiers", () => {
    expect(source).toContain('tag: "badge-size-tag"')
    expect(source).toContain('compact: "badge-size-compact"')
    expect(source).toContain('micro: "badge-size-micro"')
    expect(source).toContain("export type BadgeStructuralSize")
    expect(globals).toContain(".badge-size-tag")
    expect(globals).toContain("height: 1.625rem")
    expect(globals).toContain(".badge-size-compact")
    expect(globals).toContain("height: 1.25rem")
    expect(globals).toContain(".badge-size-micro")
    expect(globals).toContain("height: 1rem")
  })
})
