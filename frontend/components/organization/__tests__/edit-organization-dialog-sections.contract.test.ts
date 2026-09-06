import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/edit-organization-dialog-sections.tsx"), "utf8")

describe("edit-organization-dialog-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function EditOrganizationNameField")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("keeps drawer footer actions out of dialog chrome", () => {
    expect(source).toContain("export function EditOrganizationFooter")
    expect(source).toContain("className=\"flex flex-wrap justify-end gap-2\"")
    expect(source).toContain("semanticIcons.action.edit")
    expect(source).not.toContain("DialogFooter")
    expect(source).not.toContain("DialogHeader")
    expect(source).not.toContain("DialogTitle")
  })

  it("keeps dismissal on the side-panel header instead of duplicating cancel in the footer", () => {
    expect(source).not.toContain("onCancel: () => void")
    expect(source).not.toContain('{t("cancel")}')
    expect(source).toContain("onClick={onReset}")
    expect(source).toContain('type="submit"')
  })

  it("keeps optional-field guidance beside the organization description field", () => {
    expect(source).toContain('{t("orgDescOptional")}')
  })
})
