import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "lib/scheduled-scan-helpers.ts"), "utf8")

describe("scheduled-scan-helpers contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export type ScheduledScanSelectionMode = \"organization\" | \"target\"")
  })

  it("owns strict five-field Cron and UTC previews", () => {
    expect(source).toContain('expression.startsWith("@")')
    expect(source).toContain("parts.length !== 5")
    expect(source).toContain("CronExpressionParser.parse(expression)")
    expect(source).toContain('tz: "UTC"')
    expect(source).not.toContain("Intl.supportedValuesOf")
  })

  it("validates the current Workflow reference without a removed preset identity", () => {
    expect(source).toContain('return scanWorkflow ? null : "form.scanWorkflowRequired"')
    expect(source).not.toContain("selectedPresetId")
  })
})
