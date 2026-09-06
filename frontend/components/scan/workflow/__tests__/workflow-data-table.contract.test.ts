import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/workflow/workflow-data-table.tsx"), "utf8")

describe("workflow-data-table contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function WorkflowDataTable")
    expect(source).toContain("from \"@tanstack/react-table\"")
    expect(source).toContain("BusinessListDataTable")
    expect(source).not.toContain("from \"@/components/shared/data-table/unified-data-table\"")
  })
})
