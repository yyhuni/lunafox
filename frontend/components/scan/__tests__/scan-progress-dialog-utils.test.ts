import { describe, expect, it } from "vitest"

import {
  NUCLEI_POC_FAILURE_KIND,
  NUCLEI_POC_MANAGEMENT_HREF,
  buildNucleiPocFailureGuidance,
  buildScanProgressData,
} from "@/components/scan/scan-progress-dialog-utils"
import type { ScanRecord } from "@/types/scan.types"

function failedScan(overrides: Partial<ScanRecord> = {}): ScanRecord {
  return {
    id: 21,
    targetId: 5,
    plannedEngineIds: ["engine.lunafox.nuclei_vulnerability"],
    triggerType: "manual",
    inputSource: "scanSnapshot",
    createdAt: "2026-10-21T00:00:00Z",
    status: "failed",
    progress: 100,
    ...overrides,
  }
}

describe("buildScanProgressData", () => {
  it("preserves each runtime task engine ID for localized stage presentation", () => {
    const scan: ScanRecord = {
      id: 17,
      targetId: 5,
      plannedEngineIds: ["engine.lunafox.website_discovery"],
      triggerType: "manual",
      inputSource: "scanSnapshot",
      createdAt: "2026-07-10T00:00:00Z",
      status: "running",
      progress: 50,
      runtimeTasks: [{
        id: 1701,
        name: "scans/17/tasks/1701",
        stepId: "collect_urls_for_priority_targets",
        stageId: "discovery",
        engineId: "engine.lunafox.website_discovery",
        status: "running",
        order: 0,
      }],
    }

    expect(buildScanProgressData(scan).stages).toMatchObject([{
      engineId: "engine.lunafox.website_discovery",
      stage: "collect_urls_for_priority_targets",
    }])
  })

  it("builds actionable guidance only for the Nuclei POC failure kind", () => {
    const scan = failedScan({
      failure: {
        kind: NUCLEI_POC_FAILURE_KIND,
        message: "The runtime artifact required by this task is unavailable.",
      },
    })

    expect(buildScanProgressData(scan)).toMatchObject({
      failureKind: NUCLEI_POC_FAILURE_KIND,
      failureGuidance: {
        kind: NUCLEI_POC_FAILURE_KIND,
        targetHref: NUCLEI_POC_MANAGEMENT_HREF,
      },
    })
  })

  it("keeps other failure kinds on the plain raw-message path", () => {
    const scan = failedScan({
      failure: { kind: "engine_exit_failed", message: "engine exited with code 1" },
    })

    const data = buildScanProgressData(scan)
    expect(data.failureKind).toBe("engine_exit_failed")
    expect(data.failureGuidance).toBeNull()
    expect(data.errorMessage).toBe("engine exited with code 1")
  })

  it("has no guidance when the failure projection is absent", () => {
    const data = buildScanProgressData(failedScan())
    expect(data.failureGuidance).toBeNull()
    expect(data.errorMessage).toBeUndefined()
  })
})

describe("buildNucleiPocFailureGuidance", () => {
  it("returns the canonical POC href only for the exact failure kind", () => {
    expect(buildNucleiPocFailureGuidance(NUCLEI_POC_FAILURE_KIND)).toEqual({
      kind: NUCLEI_POC_FAILURE_KIND,
      targetHref: NUCLEI_POC_MANAGEMENT_HREF,
    })
    expect(buildNucleiPocFailureGuidance("")).toBeNull()
    expect(buildNucleiPocFailureGuidance(undefined)).toBeNull()
    expect(buildNucleiPocFailureGuidance("no_enabled_templates_v2")).toBeNull()
  })
})
