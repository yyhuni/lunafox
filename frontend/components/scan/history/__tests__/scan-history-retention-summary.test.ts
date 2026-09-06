import { describe, expect, it } from "vitest"
import { formatScanHistoryRetentionDuration } from "../scan-history-retention-summary"

describe("formatScanHistoryRetentionDuration", () => {
  it("formats the backend-provided retention duration in the active locale", () => {
    expect(formatScanHistoryRetentionDuration(14 * 24 * 60 * 60, "zh")).toBe("14天")
    expect(formatScanHistoryRetentionDuration(14 * 24 * 60 * 60, "en")).toBe("14d")
    expect(formatScanHistoryRetentionDuration(26 * 60 * 60, "en")).toBe("1d 2h")
  })

  it("keeps sub-day backend values readable instead of assuming days", () => {
    expect(formatScanHistoryRetentionDuration(90 * 60, "zh")).toBe("1小时 30分钟")
    expect(formatScanHistoryRetentionDuration(45, "en")).toBe("45s")
  })
})
