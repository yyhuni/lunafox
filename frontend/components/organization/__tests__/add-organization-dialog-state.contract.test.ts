import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/add-organization-dialog-state.ts"), "utf8")

describe("add-organization-dialog-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useAddOrganizationDialogState")
    expect(source).toContain("from \"react\"")
  })

  it("owns collapsed target section state without a separate clear action", () => {
    expect(source).toContain("const [targetsExpanded, setTargetsExpanded] = React.useState(false)")
    expect(source).toContain("handleToggleTargetsExpanded")
    expect(source).toContain("setTargetsExpanded((expanded) => !expanded)")
    expect(source).not.toContain("handleClearTargets")
    expect(source).not.toContain("form.setValue(\"targets\", \"\"")
    expect(source).not.toContain("hasTargetsText")
  })

  it("resets target disclosure only when the dialog closes", () => {
    expect(source).toContain("if (!nextOpen)")
    expect(source).toContain("setTargetsExpanded(false)")
  })

  it("preserves original target input line numbers for validation feedback", () => {
    expect(source).toContain("TargetValidator.parseLines(targetsText)")
    expect(source).toContain("lineNumber: result.lineNumber")
  })

  it("creates initial targets with organizationIds array payload", () => {
    expect(source).toContain("organizationIds: [newOrganization.id]")
    expect(source).not.toContain("organizationId: newOrganization.id")
  })
})
