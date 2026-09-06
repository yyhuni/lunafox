import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "types/scheduled-scan.types.ts"), "utf8")

describe("scheduled-scan.types contract", () => {
  it("keeps the UTC-only request shape and nullable trigger cursors", () => {
    expect(source).not.toContain("timeZone")
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
