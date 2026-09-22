import { describe, expect, it } from "vitest"
import { mockScheduledScans } from "@/mock/data/scheduled-scans"

const API_BASE = "http://localhost/v1"

describe("Scheduled Scan time-zone mock network boundary", () => {
  it("keeps the persisted cursor when a time-rule patch repeats its saved values", async () => {
    const scheduledScan = mockScheduledScans.find((item) => item.isEnabled && item.nextRunTime !== null)
    if (!scheduledScan || !scheduledScan.nextRunTime) {
      throw new Error("mock Scheduled Scan fixtures require an enabled task with a cursor")
    }

    const originalNextRunTime = scheduledScan.nextRunTime
    const response = await fetch(`${API_BASE}/scheduledScans/${scheduledScan.id}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name: scheduledScan.name,
        cronExpression: scheduledScan.cronExpression,
        timeZone: scheduledScan.timeZone,
        updateMask: "cronExpression,timeZone",
      }),
    })

    expect(response.status).toBe(200)
    expect(await response.json()).toMatchObject({
      cronExpression: scheduledScan.cronExpression,
      timeZone: scheduledScan.timeZone,
      nextRunTime: originalNextRunTime,
    })
  })
})
