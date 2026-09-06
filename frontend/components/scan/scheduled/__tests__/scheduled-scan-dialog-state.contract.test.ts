import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scheduled/scheduled-scan-dialog-state.ts"), "utf8")

describe("scheduled-scan-dialog-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useScheduledScanDialogState")
    expect(source).toContain("from \"react\"")
  })

  it("keeps create form and previews UTC-only", () => {
    expect(source).not.toContain("getBrowserTimeZone")
    expect(source).not.toContain("timeZone: timeZone.trim()")
    expect(source).toContain("getNextCronExecutions(cron")
    expect(source).toContain('timeZone: "UTC"')
  })
})
