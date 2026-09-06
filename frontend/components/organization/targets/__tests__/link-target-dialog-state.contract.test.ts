import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/targets/link-target-dialog-state.ts"), "utf8")

describe("link-target-dialog-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useLinkTargetDialogState")
    expect(source).toContain("from \"react\"")
  })

  it("preserves original target input line numbers for validation feedback", () => {
    expect(source).toContain("TargetValidator.parseLines(targetsText)")
    expect(source).toContain("lineNumber: result.lineNumber")
  })

  it("creates linked targets with organizationIds array payload", () => {
    expect(source).toContain("organizationIds: [organizationId]")
    expect(source).not.toContain("targets: targetList,\n        organizationId,")
  })
})
