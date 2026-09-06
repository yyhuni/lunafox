import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "types/overview.types.ts"), "utf8")

describe("overview.types contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export interface OverviewStats {")
  })
})
