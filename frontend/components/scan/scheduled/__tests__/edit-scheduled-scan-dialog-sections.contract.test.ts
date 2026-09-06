import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scheduled/edit-scheduled-scan-dialog-sections.tsx"), "utf8")

describe("edit-scheduled-scan-dialog-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function EditScheduledScanDialogHeader")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("keeps the edit schedule section free of a zone selector", () => {
    expect(source).not.toContain("ScheduledScanTimeZoneField")
    expect(source).not.toContain("time-zone")
  })

  it("does not keep a scheduled-scan-local workflow picker", () => {
    expect(source).not.toContain("EditScheduledScanWorkflowSection")
  })

})
