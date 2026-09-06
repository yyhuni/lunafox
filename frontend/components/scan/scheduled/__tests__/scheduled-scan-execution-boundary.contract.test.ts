import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const read = (relativePath: string) => readFileSync(path.resolve(process.cwd(), relativePath), "utf8")

const serviceSource = read("services/scheduled-scan.service.ts")
const scheduledTypesSource = read("types/scheduled-scan.types.ts")
const scanTypesSource = read("types/scan.types.ts")
const pageSource = read("components/scan/scheduled/scheduled-scan-page.tsx")
const columnsSource = read("components/scan/scheduled/scheduled-scan-columns.tsx")
const createSource = read("components/scan/scheduled/create-scheduled-scan-dialog.tsx")

describe("scheduled scan first-phase execution boundary", () => {
  it("keeps occurrence state private and adds no scheduling provenance to ordinary Scans", () => {
    expect(serviceSource).not.toMatch(/occurrence|triggerHistory/i)
    expect(scheduledTypesSource).not.toMatch(/ScheduledScanOccurrence|occurrenceId/)
    expect(scanTypesSource).not.toMatch(/scheduledScanId|occurrenceId|scheduledFor/)
  })

  it("adds no Run Now, scheduler setting, dashboard, or health surface", () => {
    const managementSurface = [serviceSource, pageSource, columnsSource, createSource].join("\n")
    expect(managementSurface).not.toMatch(/runNow|runScheduledScan|schedulerSetting/i)
    expect(managementSurface).not.toMatch(/schedulerDashboard|schedulerHealth/i)
    expect(serviceSource).not.toMatch(/:run|\/occurrences/)
  })

  it("preserves the existing four-step management workbench", () => {
    expect(createSource).toContain("const totalSteps = 4")
    expect(createSource).toContain("ScheduledScanScheduleStep")
    expect(createSource).toContain("InitiateScanWorkflowSelection")
    expect(createSource).toContain("InitiateScanConfigStep")
    expect(createSource).toContain("currentStep === 4")
  })
})
