import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scheduled/edit-scheduled-scan-dialog-state.ts"), "utf8")

describe("edit-scheduled-scan-dialog-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useEditScheduledScanDialogState")
    expect(source).toContain("from \"react\"")
  })

  it("initializes Edit with the UTC-only schedule contract", () => {
    expect(source).not.toContain("setTimeZone")
    expect(source).not.toContain("timeZone: timeZone.trim()")
  })
})
