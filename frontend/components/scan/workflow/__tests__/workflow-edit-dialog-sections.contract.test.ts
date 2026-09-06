import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/workflow/workflow-edit-dialog-sections.tsx"), "utf8")

describe("workflow-edit-dialog-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function WorkflowEditHeader")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/icons\"")
  })
})
