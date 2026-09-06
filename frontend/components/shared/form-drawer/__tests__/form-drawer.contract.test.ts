import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/form-drawer/form-drawer.tsx"), "utf8")
const indexSource = readFileSync(path.resolve(process.cwd(), "components/shared/form-drawer/index.ts"), "utf8")
const readmeSource = readFileSync(path.resolve(process.cwd(), "components/shared/form-drawer/README.md"), "utf8")

describe("form-drawer contract", () => {
  it("exports the shared form drawer shell", () => {
    expect(source).toContain("export function FormDrawer")
    expect(source).toContain("export function FormDrawerPanel")
    expect(indexSource).toContain("FormDrawer")
    expect(indexSource).toContain("FormDrawerPanel")
  })

  it("uses the shared sheet and form drawer style owner", () => {
    expect(source).toContain("SheetContent")
    expect(source).toContain("SheetClose")
    expect(source).toContain("formDrawerContentClassName")
    expect(source).not.toContain("sideMotion")
    expect(source).not.toContain("formDrawerBackdropClassName")
    expect(source).not.toContain("overlayClassName")
    expect(source).not.toContain("overlayBackdropBlur")
  })

  it("inherits the shared shadcn base-nova backdrop instead of a local variant", () => {
    expect(readmeSource).toContain("shared shadcn base-nova overlay")
    expect(readmeSource).not.toContain("overlayBackdropBlur={false}")
    expect(readmeSource).not.toContain("formDrawerBackdropClassName")
  })

  it("inherits the shared edge-panel first-mount handoff", () => {
    expect(readmeSource).toContain("shared Sheet primitive")
    expect(readmeSource).toContain("reduced motion")
  })

  it("keeps form drawer body and footer as owned regions", () => {
    expect(source).toContain("flex h-full min-h-0 w-full flex-col")
    expect(source).toContain("overflow-y-auto px-6 py-4")
    expect(source).toContain("border-t px-6 py-4")
    expect(source).toContain("FormDrawerFrame")
  })

  it("supports an embedded panel variant for split detail workflows", () => {
    expect(source).toContain("interface FormDrawerPanelProps")
    expect(source).toContain("onClose")
    expect(source).toContain("aria-label={tActions(\"close\")}")
    expect(readmeSource).toContain("Embedded Panel")
    expect(readmeSource).toContain("split detail workflows")
    expect(readmeSource).toContain("fills the parent column width")
  })

  it("uses the shared quick-scan title hierarchy for drawer and embedded panel titles", () => {
    expect(source).toContain("SheetTitle")
    expect(source).toContain("textRole.sectionTitle")
    expect(source).toContain('className={cn("max-w-2xl", textRole.helperText)}')
    expect(source).toContain("EdgePanelHeader")
    expect(source).toContain('variant="form"')
    expect(source).not.toContain("textRole.panelTitle")
    expect(readmeSource).toContain("textRole.sectionTitle")
    expect(readmeSource).toContain("textRole.helperText")
  })

  it("delegates form header geometry to the shared form variant", () => {
    expect(source).toContain("EdgePanelHeader")
    expect(source).toContain('leading={icon}')
    expect(source).toContain('actions={closeControl}')
    expect(readmeSource).toContain("min-h-8")
    expect(readmeSource).toContain("size-10")
  })

  it("keeps drawer and embedded panel close affordances on shared quiet icon buttons", () => {
    expect(source).toContain('showCloseButton={false}')
    expect(source).toContain('render={<Button type="button" variant="ghost" size="icon-sm" aria-label={tActions("close")} disabled={closeDisabled}/>}')
    expect(source).toContain('variant="ghost" size="icon-sm" aria-label={tActions("close")} disabled={closeDisabled} onClick={onClose}')
    expect(source).toContain("<semanticIcons.action.cancel />")
    expect(source).toContain('import { semanticIcons } from "@/components/icons"')
    expect(source).not.toContain("focus:ring-")
  })

  it("documents stable field ordering for optional expanding association controls", () => {
    for (const label of [
      "Field Ordering",
      "primary task field",
      "optional association fields",
      "must not push the primary task field",
      "Popover",
      "Command",
    ]) {
      expect(readmeSource).toContain(label)
    }
  })
})
