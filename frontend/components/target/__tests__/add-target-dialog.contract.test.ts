import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/target/add-target-dialog.tsx"), "utf8")

describe("add-target-dialog contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function AddTargetDialog")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("keeps the footer focused on the submit action", () => {
    expect(source).toContain("tDialog={t}")
    expect(source).not.toContain("tActions={tCommon}")
  })

  it("renders the creation flow in the shared side drawer shell", () => {
    expect(source).toContain("FormDrawer")
    expect(source).toContain('from "@/components/shared/form-drawer"')
    expect(source).toContain('from "@/lib/ui/overlay-styles"')
    expect(source).toContain("scanWorkbenchDrawerContentClassName")
    expect(source).toContain('bodyClassName="gap-6 px-5 py-5 sm:px-6 sm:py-6"')
    expect(source).not.toContain("DialogContent")
    expect(source).not.toContain("scrollableFormDialogContentClassName")
  })

  it("renders the target input before the optional organization picker", () => {
    const inputIndex = source.indexOf("<AddTargetInputSection")
    const organizationIndex = source.indexOf("<AddTargetOrganizationPicker")

    expect(inputIndex).toBeGreaterThan(-1)
    expect(organizationIndex).toBeGreaterThan(-1)
    expect(inputIndex).toBeLessThan(organizationIndex)
  })

  it("passes multi-organization state and clear action to the optional organization picker", () => {
    expect(source).toContain("handleToggleOrganization")
    expect(source).toContain("handleClearOrganizations")
    expect(source).toContain("orgPageSize")
    expect(source).toContain("setOrgPageSize")
    expect(source).toContain("formOrganizationIds={formData.organizationIds}")
    expect(source).not.toContain("selectedOrganizations")
    expect(source).toContain("orgPageSize={orgPageSize}")
    expect(source).toContain("setOrgPageSize={setOrgPageSize}")
    expect(source).toContain("onToggleOrganization={handleToggleOrganization}")
    expect(source).toContain("onClearOrganizations={handleClearOrganizations}")
    expect(source).not.toContain("pageSizeOptions={pageSizeOptions}")
  })
})
