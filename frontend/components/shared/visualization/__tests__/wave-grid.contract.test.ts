import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/visualization/wave-grid.tsx"), "utf8")

describe("wave-grid contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function WaveGrid")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })
})
