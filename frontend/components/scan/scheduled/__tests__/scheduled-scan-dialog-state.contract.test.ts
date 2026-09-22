import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scheduled/scheduled-scan-dialog-state.ts"), "utf8")

describe("scheduled-scan-dialog-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useScheduledScanDialogState")
    expect(source).toContain("from \"react\"")
  })

  it("prefills the browser IANA zone and uses it for create previews", () => {
    expect(source).toContain("getBrowserTimeZone")
    expect(source).toContain("timeZone: timeZone.trim()")
    expect(source).toContain("getNextCronExecutions(cron")
    expect(source).toContain("getNextCronExecutions(cron, zone")
  })
})
