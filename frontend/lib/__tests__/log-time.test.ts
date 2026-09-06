import { describe, expect, it } from "vitest"

import { formatLocalTimestampSeconds, formatLogTimestamp } from "@/lib/log-time"
import { formatScanLogLine } from "@/components/scan/scan-log-list-state"

describe("formatLogTimestamp", () => {
  it("includes the local calendar date and milliseconds", () => {
    const value = new Date(2026, 6, 24, 20, 9, 10, 123).toISOString()

    expect(formatLogTimestamp(value)).toBe("2026-07-24 20:09:10.123")
  })

  it("returns a stable placeholder for invalid timestamps", () => {
    expect(formatLogTimestamp("not-a-timestamp")).toBe("---- -- -- --:--:--.---")
  })

  it("formats task timestamps with the local calendar date and second precision", () => {
    const value = new Date(2026, 6, 24, 20, 9, 32).toISOString()

    expect(formatLocalTimestampSeconds(value)).toBe("2026-07-24 20:09:32")
    expect(formatLocalTimestampSeconds("not-a-timestamp")).toBe("---- -- -- --:--:--")
  })

  it("keeps scan log lines at local second precision", () => {
    const createdAt = new Date(2026, 6, 24, 20, 9, 10, 123).toISOString()

    expect(
      formatScanLogLine({
        id: 1,
        taskId: 7001,
        level: "info",
        content: "stage started",
        createdAt,
      }),
    ).toBe("[2026-07-24 20:09:10] [INFO] stage started")
  })
})
