import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/tools/commands/commands-data-table-state.ts"), "utf8")

describe("commands-data-table-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useCommandsDataTableState")
    expect(source).toContain("from \"react\"")
  })
})
