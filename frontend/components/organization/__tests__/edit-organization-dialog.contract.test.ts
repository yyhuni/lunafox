import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/edit-organization-dialog.tsx"), "utf8")

describe("edit-organization-dialog contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function EditOrganizationDialog")
    expect(source).toContain("export function EditOrganizationFormPanel")
    expect(source).toContain("FormDrawer")
    expect(source).toContain("FormDrawerPanel")
    expect(source).toContain("from \"react\"")
  })

  it("uses the shared form drawer instead of a centered dialog", () => {
    expect(source).toContain("from \"@/components/shared/form-drawer\"")
    expect(source).toContain("title={t(\"editTitle\")}")
    expect(source).not.toContain("description={t(\"editDesc\")}")
    expect(source).toContain("formProps={{ onSubmit: form.handleSubmit(onSubmit) }}")
    expect(source).not.toContain("DialogContent")
    expect(source).not.toContain("narrowFormDialogContentClassName")
  })

  it("shares the same form surface between drawer and split panel presentations", () => {
    expect(source).toContain("function EditOrganizationFormSurface")
    expect(source).toContain('presentation === "panel"')
    expect(source).toContain("onClose={() => handleOpenChange(false)}")
    expect(source).not.toContain("onCancel={() => handleOpenChange(false)}")
  })
})
