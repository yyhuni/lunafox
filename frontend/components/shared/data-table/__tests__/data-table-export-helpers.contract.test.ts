import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/data-table/data-table-export-helpers.ts"), "utf8")

describe("data-table-export-helpers contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function buildExportOptions")
    expect(source).toContain("from \"@/types/data-table.types\"")
  })
})
