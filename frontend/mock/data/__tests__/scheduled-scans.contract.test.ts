import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"
import { getMockScheduledScans } from "@/mock/data/scheduled-scans"

const source = readFileSync(path.resolve(process.cwd(), "mock/data/scheduled-scans.ts"), "utf8")
const handlerSource = readFileSync(path.resolve(process.cwd(), "mock/handlers/index.ts"), "utf8")

describe("scheduled-scans contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function getMockScheduledScans")
    expect(source).toContain("from '@/types/scheduled-scan.types'")
    expect(source).toContain("params?.organizationId")
    expect(source).toContain("params?.targetId")
  })

  it("keeps every scheduled fixture on the canonical explicit Step envelope", () => {
    for (const schedule of getMockScheduledScans().scheduledScans) {
      const configuration = schedule.configuration
      if (!configuration || typeof configuration === "string" || Array.isArray(configuration)) {
        throw new Error("scheduled scan fixture configuration must be a canonical object")
      }
      const steps = configuration.steps
      expect(steps).toBeDefined()
      expect(Object.keys(steps as Record<string, unknown>).length).toBeGreaterThan(0)
      for (const [stepId, rawStep] of Object.entries(steps as Record<string, unknown>)) {
        expect(rawStep).toBeTypeOf("object")
        const step = rawStep as Record<string, unknown>
        expect(typeof step.enabled, stepId).toBe("boolean")
        if (step.enabled === false) {
          expect(Object.keys(step)).toEqual(["enabled"])
        } else {
          expect(step.engineConfig).toBeTypeOf("object")
          expect(Object.keys(step.engineConfig as Record<string, unknown>).length).toBeGreaterThan(0)
        }
      }
    }
  })

  it("keeps UTC cursors and null cursors for disabled schedules", () => {
    for (const schedule of getMockScheduledScans().scheduledScans) {
      if (!schedule.isEnabled) {
        expect(schedule.nextRunTime).toBeNull()
      }
    }
  })

  it("gives every Scheduled Scan fixture an explicit supported input source", () => {
    for (const schedule of getMockScheduledScans().scheduledScans) {
      expect(["scanSnapshot", "targetInventory"]).toContain(schedule.inputSource)
    }
  })

  it("uses opaque page tokens for scheduled scan continuation", () => {
    const firstPage = getMockScheduledScans({ pageSize: 2 })
    const secondPage = getMockScheduledScans({ pageSize: 2, pageToken: firstPage.nextPageToken })

    expect(firstPage.nextPageToken).toBe("mock-scheduled-scan-page-2")
    expect(secondPage.scheduledScans[0]?.id).toBe(3)
    expect(secondPage.totalSize).toBe(firstPage.totalSize)
  })

  it("rejects retired time-zone fields from mock producers", () => {
    expect(handlerSource).toContain('"timeZone" in (body as Record<string, unknown>)')
    expect(handlerSource).toContain("schedules use UTC")
  })
})
