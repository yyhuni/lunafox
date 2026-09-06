import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-horizontal-resize.ts"), "utf8")

describe("use-horizontal-resize contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useHorizontalResize")
    expect(source).toContain("from \"react\"")
  })
})
