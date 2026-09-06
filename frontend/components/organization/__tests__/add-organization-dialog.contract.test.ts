import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/add-organization-dialog.tsx"), "utf8")

describe("add-organization-dialog contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function AddOrganizationDialog")
    expect(source).toContain("FormDrawer")
    expect(source).toContain("from \"react\"")
  })

  it("uses the shared form drawer instead of a centered dialog", () => {
    expect(source).toContain("from \"@/components/shared/form-drawer\"")
    expect(source).toContain("title={t(\"addTitle\")}")
    expect(source).toContain("description={t(\"addDesc\")}")
    expect(source).toContain("formProps={{ onSubmit: form.handleSubmit(onSubmit) }}")
    expect(source).toContain("closeDisabled={isSubmitting}")
    expect(source).not.toContain("DialogContent")
    expect(source).not.toContain("DialogTrigger")
    expect(source).not.toContain("scrollableFormDialogContentClassName")
  })

  it("wires optional target section state into the dialog sections", () => {
    expect(source).toContain("targetsExpanded")
    expect(source).toContain("handleToggleTargetsExpanded")
    expect(source).not.toContain("targetCount={targetValidation.count}")
    expect(source).not.toContain("hasTargetsText")
    expect(source).not.toContain("handleClearTargets")
  })
})
