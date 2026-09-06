import { describe, expect, it } from "vitest"

import {
  buildTaskProgressLogsByTaskId,
  getRuntimeTaskPresentationStatus,
  getRuntimeTaskDurationSeconds,
  toRuntimeTaskItems,
} from "@/components/scan/history/scan-runtime-detail-utils"
import type { ScanLog } from "@/types/scan.types"
import type { ScanRecord } from "@/types/scan.types"

describe("buildTaskProgressLogsByTaskId", () => {
  it("distinguishes interrupted tasks from cancelled tasks that never started", () => {
    expect(getRuntimeTaskPresentationStatus({ status: "cancelled", startedAt: "2026-06-19T01:03:00Z" })).toBe("interrupted")
    expect(getRuntimeTaskPresentationStatus({ status: "cancelled" })).toBe("not_started")
    expect(getRuntimeTaskPresentationStatus({ status: "pending" })).toBe("pending")
  })

  it("groups explicit progress entries by task identity and sorts each task timeline", () => {
    const logs: ScanLog[] = [
      {
        id: 3,
        taskId: 7001,
        level: "info",
        content: "third event",
        createdAt: "2026-06-19T01:02:05Z",
      },
      {
        id: 1,
        taskId: 7001,
        level: "info",
        content: "first event",
        createdAt: "2026-06-19T01:02:03Z",
      },
      {
        id: 2,
        taskId: 7002,
        level: "warning",
        content: "other task",
        createdAt: "2026-06-19T01:02:04Z",
      },
    ]

    const grouped = buildTaskProgressLogsByTaskId(logs)

    expect(grouped.get(7001)?.map((log) => log.id)).toEqual([1, 3])
    expect(grouped.get(7002)?.map((log) => log.content)).toEqual(["other task"])
    expect(grouped.size).toBe(2)
  })

  it("keeps failure summaries scoped to failed tasks", () => {
    const scan: ScanRecord = {
      id: 7,
      targetId: 1,
      plannedEngineIds: ["engine.lunafox.subdomain_discovery", "engine.lunafox.nuclei_vulnerability"],
      triggerType: "manual",
      inputSource: "scanSnapshot",
      createdAt: "2026-06-19T01:02:00Z",
      status: "failed",
      failure: { kind: "engine_execution_failed", message: "nuclei exited with code 1" },
      progress: 50,
      runtimeTasks: [
        {
          id: 7001,
          name: "scans/1/tasks/7001",
          stepId: "subdomain_discovery",
          stageId: "discovery",
          engineId: "engine.lunafox.subdomain_discovery",
          status: "succeeded",
          order: 0,
          startedAt: "2026-06-19T01:02:03Z",
          duration: 45,
        },
        {
          id: 7002,
          name: "scans/1/tasks/7002",
          stepId: "nuclei_vulnerability",
          stageId: "risk",
          engineId: "engine.lunafox.nuclei_vulnerability",
          status: "failed",
          order: 1,
          startedAt: "2026-06-19T01:03:00Z",
          duration: 12,
          error: "template execution timed out",
          failureKind: "engine_execution_failed",
		  failureDetail: "Inspect the Agent execution storage.",
        },
      ],
    }

    const tasks = toRuntimeTaskItems(
      scan.runtimeTasks,
      scan,
      new Map([
        ["engine.lunafox.subdomain_discovery", "Subdomain Discovery"],
        ["engine.lunafox.nuclei_vulnerability", "Nuclei vulnerability scan"],
      ]),
      new Map([
        ["engine.lunafox.subdomain_discovery", "Discover subdomains through reconnaissance, optional dictionary brute force, and DNS resolution."],
        ["engine.lunafox.nuclei_vulnerability", "Run the controlled Nuclei Website vulnerability scan."],
      ]),
    )

    expect(tasks.find((task) => task.id === 7001)?.detail).toBe("Discover subdomains through reconnaissance, optional dictionary brute force, and DNS resolution.")
    expect(tasks.find((task) => task.id === 7002)?.detail).toBe("Run the controlled Nuclei Website vulnerability scan.")
    expect(tasks.find((task) => task.id === 7001)?.failureSummary).toBeUndefined()
    expect(tasks.find((task) => task.id === 7002)?.failureSummary).toBe("template execution timed out")
    expect(tasks.find((task) => task.id === 7002)?.failureDetail).toBe("Inspect the Agent execution storage.")
  })

  it("calculates running task duration from its UTC start time", () => {
    const duration = getRuntimeTaskDurationSeconds(
      {
        id: 7003,
        title: "Nuclei Scan",
        status: "running",
        detail: "Detect known vulnerabilities across discovered target surfaces.",
        startedAt: "2026-06-19T01:03:00Z",
        order: 2,
      },
      new Date("2026-06-19T01:04:12Z")
    )

    expect(duration).toBe(72)
  })

  it("keeps completed task duration from the backend result", () => {
    const duration = getRuntimeTaskDurationSeconds(
      {
        id: 7004,
        title: "Screenshot",
        status: "succeeded",
        detail: "Capture a screenshot for the discovered website.",
        startedAt: "2026-06-19T01:03:00Z",
        duration: 53,
        order: 3,
      },
      new Date("2026-06-19T01:04:12Z")
    )

    expect(duration).toBe(53)
  })
})
