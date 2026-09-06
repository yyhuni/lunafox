import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/targets/link-target-dialog-sections.tsx"), "utf8")

describe("link-target-dialog-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function LinkTargetInputSection")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("keeps drawer footer actions out of dialog chrome", () => {
    expect(source).toContain("export function LinkTargetDialogFooter")
    expect(source).toContain("className=\"flex justify-end gap-2\"")
    expect(source).toContain("semanticIcons.action.add")
    expect(source).not.toContain("from \"@/components/ui/dialog\"")
    expect(source).not.toContain("<DialogFooter")
    expect(source).not.toContain("<DialogHeader")
    expect(source).not.toContain("<DialogTitle")
  })

  it("renders target validation examples with original input line numbers", () => {
    expect(source).toContain("lineNumber: target.lineNumber")
    expect(source).not.toContain("line: targetValidation.invalid[0].index + 1")
  })

  it("uses the shared bulk line validation input for target entry guidance", () => {
    expect(source).toContain("BulkLineValidationInput")
    expect(source).toContain("lineNumberedTextareaResponsiveViewportClassName")
    expect(source).toContain("viewportClassName={lineNumberedTextareaResponsiveViewportClassName}")
    expect(source).toContain("helper={t(\"targetHelper\")}")
    expect(source).toContain("example={t(\"targetExample\")}")
    expect(source).not.toContain("<LineNumberedTextarea")
    expect(source).not.toContain("<FormDescription")
  })

  it("keeps the organization label text-only while retaining the selected organization icon", () => {
    const labelStart = source.indexOf("<Label>")
    const labelEnd = source.indexOf("</Label>", labelStart)
    const labelBlock = source.slice(labelStart, labelEnd)

    expect(labelStart).toBeGreaterThanOrEqual(0)
    expect(labelBlock).toContain('{t("organizationLabel")}')
    expect(labelBlock).not.toContain("OrganizationIcon")
    expect(source).toContain('<OrganizationIcon className="h-4 text-muted-foreground w-4" />')
  })
})
