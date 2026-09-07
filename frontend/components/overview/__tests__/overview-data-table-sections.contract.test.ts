import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/overview/overview-data-table-sections.tsx"), "utf8")

describe("overview-data-table-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function OverviewDataDialogs")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })
})
