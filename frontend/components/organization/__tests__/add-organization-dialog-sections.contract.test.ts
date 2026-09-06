import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/add-organization-dialog-sections.tsx"), "utf8")

describe("add-organization-dialog-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function AddOrganizationNameField")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("keeps drawer footer actions owned by shared buttons instead of dialog footer chrome", () => {
    expect(source).toContain("export function AddOrganizationFooter")
    expect(source).toContain("className=\"flex justify-end gap-2\"")
    expect(source).toContain("semanticIcons.action.add")
    expect(source).not.toContain("DialogFooter")
    expect(source).not.toContain("DialogHeader")
    expect(source).not.toContain("DialogTitle")
  })

  it("keeps optional target entry as a low-emphasis collapsible section", () => {
    expect(source).toContain("isTargetsExpanded")
    expect(source).toContain("onToggleTargetsExpanded")
    expect(source).toContain("ChevronRight")
    expect(source).toContain('className="mt-2 gap-3 border-t border-border/60 pt-4"')
    expect(source).toContain("aria-expanded={isTargetsExpanded}")
    expect(source).toContain("justify-start gap-2")
    expect(source).toContain('className="flex min-w-0 items-center gap-2"')
    expect(source).toContain('"text-muted-foreground transition-colors group-hover:text-foreground"')
    expect(source).toContain("group-hover:text-foreground group-data-[panel-open]:rotate-90")
    expect(source).toContain('data-panel-open={isTargetsExpanded ? "" : undefined}')
    expect(source).toContain('group-data-[panel-open]:rotate-90')
    expect(source).toContain("t(\"targetHelp\")")
    expect(source).toContain("BulkLineValidationInput")
    expect(source).toContain("lineIssues")
    expect(source).toContain("blockingIssueCount")
    expect(source).not.toContain("lineNumberedTextareaCompactViewportClassName")
    expect(source).not.toContain("lineNumberedTextareaTallViewportClassName")
  })

  it("keeps the optional target header free of a separate clear action", () => {
    expect(source).not.toContain("hasTargetsText")
    expect(source).not.toContain("onClearTargets")
    expect(source).not.toContain("t(\"clearTargets\")")
  })

  it("keeps examples and recognized target count outside the textarea placeholder", () => {
    expect(source).toContain("placeholder={t(\"targetsPlaceholder\")}")
    expect(source).toContain("t(\"targetExample\")")
    expect(source).toContain("validSummary={t(\"targetValidSummary\", { count: targetValidation.count })}")
  })

  it("renders target validation feedback with original input line numbers", () => {
    expect(source).toContain("lineNumber: target.lineNumber")
    expect(source).toContain("line: target.lineNumber")
    expect(source).not.toContain("lineNumber: target.index + 1")
    expect(source).not.toContain("line: target.index + 1")
  })

  it("keeps the submit label stable regardless of optional target count", () => {
    expect(source).toContain("{t(\"create\")}")
    expect(source).not.toContain("targetCount > 0")
    expect(source).not.toContain("createWithTargets")
  })
})
