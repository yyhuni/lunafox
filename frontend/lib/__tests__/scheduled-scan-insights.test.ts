import { describe, expect, it } from "vitest"

import {
  buildScheduledScanInsights,
} from "@/lib/scheduled-scan-insights"

import type { ScheduledScan } from "@/types/scheduled-scan.types"

const scan = (overrides: Partial<ScheduledScan>): ScheduledScan => ({
  id: overrides.id ?? 1,
  name: overrides.name ?? "scheduledScans/1",
  displayName: overrides.displayName ?? "Daily Scan",
  scanWorkflow: overrides.scanWorkflow ?? "subdomain_discovery",
  configuration: overrides.configuration,
  organization: overrides.organization ?? null,
  organizationId: overrides.organizationId ?? 1,
  organizationName: overrides.organizationName ?? "Acme",
  target: overrides.target ?? null,
  targetId: overrides.targetId ?? null,
  targetName: overrides.targetName ?? null,
  scanMode: overrides.scanMode ?? "organization",
  inputSource: overrides.inputSource ?? "scanSnapshot",
  cronExpression: overrides.cronExpression ?? "0 2 * * *",
  isEnabled: overrides.isEnabled ?? true,
  nextRunTime: overrides.nextRunTime ?? null,
  lastRunTime: overrides.lastRunTime ?? null,
  runCount: overrides.runCount ?? 0,
  successfulHandoffCount: overrides.successfulHandoffCount ?? 0,
  failedHandoffCount: overrides.failedHandoffCount ?? 0,
  createdAt: overrides.createdAt ?? "2024-12-01T00:00:00Z",
  updatedAt: overrides.updatedAt ?? "2024-12-01T00:00:00Z",
})

describe("scheduled-scan-insights", () => {
  it("builds schedule summary and timeline from scan fields", () => {
    const insights = buildScheduledScanInsights(
      [
        scan({ id: 1, name: "Hourly", cronExpression: "0 * * * *", nextRunTime: "2024-12-30T02:00:00Z" }),
        scan({ id: 2, name: "Daily", cronExpression: "0 2 * * *", nextRunTime: "2024-12-30T03:00:00Z" }),
        scan({ id: 3, name: "Paused Weekly", cronExpression: "0 3 * * 0", isEnabled: false, nextRunTime: null }),
      ],
      new Date("2024-12-30T00:00:00Z")
    )

    expect(insights.summary).toMatchObject({
      todayRuns: 2,
      next24hRuns: 2,
      enabledCount: 2,
      pausedCount: 1,
    })
    expect(insights.timeline.map((item) => item.name)).toEqual(["Hourly", "Daily"])
  })
})
