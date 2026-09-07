import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/workflow/workflow-data-table-state.ts"), "utf8")

describe("workflow-data-table-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useWorkflowDataTableState")
    expect(source).toContain("from \"react\"")
  })
})
