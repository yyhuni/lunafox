import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/overview/overview-data-table.tsx"), "utf8")

describe("overview-data-table contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function OverviewDataTable")
    expect(source).toContain("from \"./overview-data-table-sections\"")
  })
})
