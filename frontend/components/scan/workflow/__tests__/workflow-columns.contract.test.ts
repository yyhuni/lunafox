import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/workflow/workflow-columns.tsx"), "utf8")

describe("workflow-columns contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function createWorkflowColumns")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses shared status helper instead of chart utility for feature status icons", () => {
    expect(source).toContain('from "@/lib/status-config"')
    expect(source).toContain("getStatusToneTextClass")
    expect(source).not.toContain("text-chart-4")
  })

  it("uses the approved compact icon button size for row actions", () => {
    expect(source).toContain("DenseRowActionMenu")
    expect(source).toContain("ariaLabel={t.actions.openMenu}")
    expect(source).not.toContain("flex h-8 p-0 w-8")
  })
})
