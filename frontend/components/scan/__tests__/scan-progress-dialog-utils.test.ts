import { describe, expect, it } from "vitest"

import { buildScanProgressData } from "@/components/scan/scan-progress-dialog-utils"
import type { ScanRecord } from "@/types/scan.types"

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
})
