import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/pixel-blast.tsx"), "utf8")

describe("pixel-blast contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from 'react'")
  })

  it("uses theme-owned visible colors instead of a baked-in palette", () => {
    expect(source).toContain("var(--pixel-blast-color)")
    expect(source).toContain("resolveCssColor")
    expect(source).not.toContain("#B19EEF")
  })
})
