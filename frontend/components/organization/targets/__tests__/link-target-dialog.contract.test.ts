import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/targets/link-target-dialog.tsx"), "utf8")

describe("link-target-dialog contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function LinkTargetDialog")
    expect(source).toContain("export function LinkTargetFormPanel")
    expect(source).toContain("FormDrawer")
    expect(source).toContain("FormDrawerPanel")
    expect(source).toContain("from \"react\"")
  })

  it("uses the shared form drawer instead of a centered dialog", () => {
    expect(source).toContain("from \"@/components/shared/form-drawer\"")
    expect(source).toContain("title={t(\"title\")}")
    expect(source).toContain("description={t(\"description\", { name: organizationName })}")
    expect(source).toContain("formProps={{ onSubmit: form.handleSubmit(onSubmit) }}")
    expect(source).not.toContain("DialogContent")
    expect(source).not.toContain("DialogTrigger")
    expect(source).not.toContain("scrollableFormDialogContentClassName")
  })

  it("shares the same target form surface between drawer and split panel presentations", () => {
    expect(source).toContain("function LinkTargetFormSurface")
    expect(source).toContain('presentation === "panel"')
    expect(source).toContain("onClose={() => handleOpenChange(false)}")
  })
})
