import { describe, expect, it } from "vitest"
import { mockScheduledScans } from "@/mock/data/scheduled-scans"

const API_BASE = "http://localhost/v1"

interface OverviewResponse {
  asOfTime: string
  enabledScheduledScanCount: number
  pausedScheduledScanCount: number
  todayScheduledScanCount: number
  next24HoursScheduledScanCount: number
  upcomingScheduledScans: Array<{
    name: string
    organization: string | null
    target: string | null
    nextRunTime: string
  }>
}

async function getOverview() {
  const response = await fetch(`${API_BASE}/scheduledScans:summarize`)
  return { response, body: await response.json() as OverviewResponse }
}

describe("Scheduled Scan overview mock network boundary", () => {
  it("serves the canonical compact overview independently of the List route", async () => {
    const { response, body } = await getOverview()

    expect(response.status).toBe(200)
    expect(body).toMatchObject({
      asOfTime: expect.any(String),
      enabledScheduledScanCount: expect.any(Number),
      pausedScheduledScanCount: expect.any(Number),
      todayScheduledScanCount: expect.any(Number),
      next24HoursScheduledScanCount: expect.any(Number),
    })
    expect(body.enabledScheduledScanCount + body.pausedScheduledScanCount).toBe(mockScheduledScans.length)
    expect(body.upcomingScheduledScans).toHaveLength(Math.min(5, body.enabledScheduledScanCount))
    expect(body.upcomingScheduledScans).toEqual(
      [...body.upcomingScheduledScans].sort((left, right) => (
        Date.parse(left.nextRunTime) - Date.parse(right.nextRunTime)
        || left.name.localeCompare(right.name)
      ))
    )
    for (const scheduledScan of body.upcomingScheduledScans) {
      expect(Number(scheduledScan.target !== null) + Number(scheduledScan.organization !== null)).toBe(1)
    }
  })

  it("rejects retired time-zone query parameters", async () => {
    const responses = await Promise.all([
      fetch(`${API_BASE}/scheduledScans:summarize?timeZone=%20%20`),
      fetch(`${API_BASE}/scheduledScans:summarize?timeZone=Local`),
      fetch(`${API_BASE}/scheduledScans:summarize?timeZone=Not%2FAZone`),
    ])

    for (const response of responses) {
      expect(response.status).toBe(400)
      expect(await response.json()).toMatchObject({ error: { code: "INVALID_ARGUMENT" } })
    }
  })

  it("keeps the response bounded and reflects empty and paused task sets", async () => {
    const enabled = mockScheduledScans.find((scheduledScan) => scheduledScan.isEnabled)
    if (!enabled) throw new Error("mock Scheduled Scan fixtures require an enabled task")

    for (let id = 100; id < 106; id += 1) {
      mockScheduledScans.push({
        ...enabled,
        id,
        name: `scheduledScans/${id}`,
        displayName: `Additional mock task ${id}`,
        isEnabled: true,
        nextRunTime: new Date(Date.now() + (id - 90) * 60_000).toISOString(),
      })
    }

    const bounded = await getOverview()
    expect(bounded.response.status).toBe(200)
    expect(bounded.body.upcomingScheduledScans).toHaveLength(5)

    enabled.isEnabled = false
    enabled.nextRunTime = null
    const paused = await getOverview()
    expect(paused.body.enabledScheduledScanCount).toBe(bounded.body.enabledScheduledScanCount - 1)
    expect(paused.body.pausedScheduledScanCount).toBe(bounded.body.pausedScheduledScanCount + 1)
    expect(paused.body.upcomingScheduledScans.map((scheduledScan) => scheduledScan.name)).not.toContain(enabled.name)

    mockScheduledScans.splice(0, mockScheduledScans.length)
    const empty = await getOverview()
    expect(empty.response.status).toBe(200)
    expect(empty.body).toMatchObject({
      enabledScheduledScanCount: 0,
      pausedScheduledScanCount: 0,
      todayScheduledScanCount: 0,
      next24HoursScheduledScanCount: 0,
      upcomingScheduledScans: [],
    })
  })
})
