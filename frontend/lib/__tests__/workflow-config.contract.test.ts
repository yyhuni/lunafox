import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "lib/workflow-config.ts"), "utf8")

describe("workflow-config contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("getScanWorkflowIcon")
    expect(source).toContain("from 'js-yaml'")
    expect(source).toContain("export function parseWorkflowConfiguration")
    expect(source).toContain("export function serializeWorkflowConfiguration")
  })
})
