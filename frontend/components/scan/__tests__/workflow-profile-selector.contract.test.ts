import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/workflow-profile-selector.tsx"), "utf8")

describe("workflow-profile-selector contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function WorkflowProfileSelector")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })
})
