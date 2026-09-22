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

  it("keeps the edit schedule section on the saved zone with a live preview", () => {
    expect(source).toContain("ScheduledScanTimeZoneField")
    expect(source).toContain("edit-scheduled-scan-time-zone")
    expect(source).toContain("getNextExecutions(cronExpression, timeZone)")
  })

  it("does not keep a scheduled-scan-local workflow picker", () => {
    expect(source).not.toContain("EditScheduledScanWorkflowSection")
  })

})
