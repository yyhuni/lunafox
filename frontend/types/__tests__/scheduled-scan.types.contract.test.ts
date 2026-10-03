import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "types/scheduled-scan.types.ts"), "utf8")

describe("scheduled-scan.types contract", () => {
  it("keeps an explicit IANA time zone in the resource and request shape", () => {
    expect(source).toContain("timeZone: string // IANA time zone used to evaluate the Cron rule")
    expect(source).toContain("timeZone?: string")
    expect(source).toContain("nextRunTime: string | null // Persisted next trigger time; null while disabled")
    expect(source).toContain("lastRunTime: string | null // Most recent committed trigger-attempt time")
    expect(source).toContain("runCount: number // Committed trigger-attempt count")
    expect(source).toContain("successfulHandoffCount: number // Completed schedule-to-Scan handoff count")
    expect(source).toContain("failedHandoffCount: number // Observed non-complete schedule-to-Scan handoff count")
  })

  it("requires the shared Scan input source for persisted schedules and creation", () => {
    expect(source).toContain("import type { ScanInputSource } from '@/types/scan.types'")
    expect(source).toContain("inputSource: ScanInputSource")
    expect(source).toContain("inputSource?: ScanInputSource")
  })
})

describe("scheduled-scan last-handoff-failure summary contract", () => {
  it("declares the closed public cause enum and the paired nullable summary fields", () => {
    expect(source).toContain("lastHandoffFailureCause: ScheduledScanHandoffFailureCause | null")
    expect(source).toContain("lastHandoffFailureTime: string | null")
    for (const cause of [
      "WORKFLOW_UNAVAILABLE",
      "AGENT_NOT_FOUND",
      "CONFIG_RESOURCE_UNAVAILABLE",
      "ENGINE_UNAVAILABLE",
      "TARGET_UNAVAILABLE",
      "INTERNAL_UNAVAILABLE",
    ]) {
      expect(source).toContain(`'${cause}'`)
    }
    expect(source).toContain("export function isScheduledScanHandoffFailureCause")
  })
})

describe("scheduled-scan occurrence history contract", () => {
  it("declares the closed status enum with the additive-only guard", () => {
    expect(source).toContain("export const SCHEDULED_SCAN_OCCURRENCE_STATUSES")
    expect(source).toContain("export type ScheduledScanOccurrenceStatus")
    expect(source).toContain("export function isScheduledScanOccurrenceStatus")
    for (const status of ["PENDING", "DISPATCHING", "RETRYING", "SUCCEEDED", "FAILED"]) {
      expect(source).toContain(`'${status}'`)
    }
  })

  it("projects server-derived rows whose duration exists only on succeeded runs", () => {
    expect(source).toContain("export interface ScheduledScanOccurrence {")
    expect(source).toContain("status: ScheduledScanOccurrenceStatus")
    expect(source).toContain("durationMs: number | null")
    expect(source).toContain("export interface GetScheduledScanOccurrencesResponse {")
    expect(source).toContain("statusCounts: ScheduledScanOccurrenceStatusCounts")
  })
})
