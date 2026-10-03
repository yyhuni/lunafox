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

// The only approved occurrence surface is the read-only history projection
// (add-scheduled-scan-occurrence-history): one GET endpoint, one drawer tab.
// Every other occurrence write/provenance surface stays forbidden.
const approvedOccurrenceSurface = read("components/scan/scheduled/scheduled-scan-occurrences-tab.tsx")
const scheduledScanHooksSource = read("hooks/use-scheduled-scans.ts")

describe("scheduled scan execution boundary", () => {
  it("exposes occurrence state only through the approved read-only history projection", () => {
    // The drawer tab reaches the projection through the shared hook; the hook
    // itself exposes exactly the approved read with no polling attached.
    expect(approvedOccurrenceSurface).toContain("useScheduledScanOccurrences")
    expect(approvedOccurrenceSurface).not.toMatch(/from ['"]@\/services\//)
    expect(scheduledScanHooksSource).toContain("export function useScheduledScanOccurrences")
    const occurrenceHookSource = scheduledScanHooksSource.slice(
      scheduledScanHooksSource.indexOf("export function useScheduledScanOccurrences")
    )
    expect(occurrenceHookSource.slice(0, occurrenceHookSource.indexOf("\n}\n", 800))).not.toContain("refetchInterval")
    // The service exposes exactly one approved occurrence read; history
    // phrases in comments stay allowed, but no trigger-history or write
    // surface may appear.
    expect(serviceSource).toContain("async function getScheduledScanOccurrences")
    expect(serviceSource).not.toMatch(/triggerHistory/i)
    expect(serviceSource).not.toMatch(/(create|update|delete|run)Occurrence/i)
    expect(scheduledTypesSource).toContain("ScheduledScanOccurrence")
    // Ordinary Scans still carry no scheduling provenance.
    expect(scanTypesSource).not.toMatch(/scheduledScanId|occurrenceId|scheduledFor/)
  })

  it("adds no Run Now, scheduler setting, dashboard, or health surface", () => {
    const managementSurface = [serviceSource, pageSource, columnsSource, createSource].join("\n")
    expect(managementSurface).not.toMatch(/runNow|runScheduledScan|schedulerSetting/i)
    expect(managementSurface).not.toMatch(/schedulerDashboard|schedulerHealth/i)
    // The only occurrences route is the approved read-only list, never a verb.
    expect(serviceSource).not.toMatch(/:run|post<[^>]*>[^)]*occurrences|patch<[^>]*>[^)]*occurrences|delete<[^>]*>[^)]*occurrences/i)
  })

  it("preserves the existing four-step management workbench", () => {
    expect(createSource).toContain("const totalSteps = 4")
    expect(createSource).toContain("ScheduledScanScheduleStep")
    expect(createSource).toContain("InitiateScanWorkflowSelection")
    expect(createSource).toContain("InitiateScanConfigStep")
    expect(createSource).toContain("currentStep === 4")
  })
})
