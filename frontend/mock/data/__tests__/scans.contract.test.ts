import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"
import { getMockScanById, getMockScanLogs, getMockScans } from "@/mock/data/scans"

const source = readFileSync(path.resolve(process.cwd(), "mock/data/scans.ts"), "utf8")

function sourceBetween(startMarker: string, endMarker: string) {
  const start = source.indexOf(startMarker)
  const end = source.indexOf(endMarker)

  expect(start).toBeGreaterThanOrEqual(0)
  expect(end).toBeGreaterThan(start)

  return source.slice(start, end)
}

describe("scans contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function getMockScans")
    expect(source).toContain("from '@/types/scan.types'")
  })

  it("keeps execution-node details out of scan list mock data", () => {
    const listData = sourceBetween("export const mockScans", "const mockScanRuntimeDetailsById")

    expect(listData).not.toContain("agentName:")
    expect(listData).not.toContain("yangyangdeMacBook-Pro-220.local-bf701d.cluster-node-central-03-runtime-worker")
    expect(listData).not.toContain("worker-02")
    expect(source).toContain("const mockScanRuntimeDetailsById")
    expect(source).toContain("agentName: 'scan-agent-02.example.internal'")
  })

  it("keeps scan history mock rows representative for table width pressure", () => {
    const listData = sourceBetween("export const mockScans", "const mockScanRuntimeDetailsById")

    expect(source).toContain("displayName:")
    expect(source).toContain("globalfinance.test.globalfinance.globalfinance.globalfinance.com")
    expect(source).toContain("plannedEngineIds: ['engine.lunafox.subdomain_discovery', 'engine.lunafox.web_crawling', 'engine.lunafox.nuclei_vulnerability']")
    expect(listData).not.toContain("engineNames:")
    expect(source).not.toContain("workflowNames:")
    expect(source).toContain("subdomainsCount: 0")
    expect(source).toContain("vulnsTotal: 23")
  })

  it("binds scan history mocks to the available default scan workflow", () => {
    const listData = sourceBetween("export const mockScans", "const mockScanRuntimeDetailsById")

    expect(listData).toContain("scanWorkflow: 'default'")
    expect(listData).not.toContain("scanWorkflow: 'subdomain_discovery'")
  })

  it("gives every Scan fixture an explicit supported input source", () => {
    for (const scan of getMockScans({ pageSize: 100 }).results) {
      expect(["scanSnapshot", "targetInventory"]).toContain(scan.inputSource)
    }
  })

  it("keeps runtime detail mocks dense enough to exercise task-level scrolling", () => {
    const scan = getMockScanById(1)
    const firstPage = getMockScanLogs(1, { pageSize: 20 })
    const secondPage = getMockScanLogs(1, { pageSize: 20, pageToken: firstPage.nextPageToken })

    expect(scan?.runtimeTasks?.length).toBeGreaterThanOrEqual(6)
    expect(firstPage.results.length).toBe(20)
    expect(firstPage.nextPageToken).toBeTruthy()
    expect(secondPage.results.length).toBeGreaterThanOrEqual(20)
    expect(getMockScanLogs(1).results.length).toBeGreaterThanOrEqual(50)
    expect(firstPage.results.filter((log) => log.taskId === 1001).length).toBeGreaterThanOrEqual(8)
    for (const task of scan?.runtimeTasks ?? []) {
      expect(getMockScanLogs(1).results.filter((log) => log.taskId === task.id).length).toBeGreaterThanOrEqual(8)
    }
  })

  it("assigns every task progress log mock to a runtime task", () => {
    const scan = getMockScanById(1)
    const taskIds = new Set(scan?.runtimeTasks?.map((task) => task.id))

    expect(taskIds.size).toBeGreaterThan(0)
    for (const log of getMockScanLogs(1).results) {
      expect(taskIds).toContain(log.taskId)
    }
  })

  it("keeps failure summaries only on failed runtime task mocks", () => {
    const scan = getMockScanById(4)

    const failedTasks = scan?.runtimeTasks?.filter((task) => task.status === "failed") ?? []
    const nonFailedTasksWithErrors = scan?.runtimeTasks?.filter((task) => task.status !== "failed" && (task.error || task.failureKind)) ?? []

    expect(failedTasks).toHaveLength(1)
    expect(scan?.plannedEngineIds).toEqual([
      "engine.lunafox.subdomain_discovery",
      "engine.lunafox.web_crawling",
      "engine.lunafox.nuclei_vulnerability",
    ])
    expect(failedTasks[0]).toMatchObject({
      id: 4003,
      error: expect.stringContaining("connection timeout"),
      failureKind: "engine_execution_failed",
    })
    expect(nonFailedTasksWithErrors).toEqual([])
  })

  it("keeps runtime detail configuration explicit without disabled draft payloads", () => {
    const configuration = getMockScanById(1)?.configuration
    if (!configuration || typeof configuration === "string" || Array.isArray(configuration)) {
      throw new Error("scan fixture configuration must be a canonical object")
    }
    const steps = configuration.steps as Record<string, unknown> | undefined
    expect(steps).toBeDefined()
    expect(Object.keys(steps ?? {}).length).toBeGreaterThan(0)
    for (const rawStep of Object.values(steps ?? {})) {
      const step = rawStep as Record<string, unknown>
      expect(typeof step.enabled).toBe("boolean")
      if (step.enabled === false) expect(Object.keys(step)).toEqual(["enabled"])
      if (step.enabled === true) expect(step.engineConfig).toBeTypeOf("object")
    }
  })

  it("keeps a container-exit failure mock for runtime detail row layout regressions", () => {
    const scan = getMockScanById(9)
    const logs = getMockScanLogs(9)
    const failedTask = scan?.runtimeTasks?.find((task) => task.status === "failed")

    expect(scan?.target?.name).toBe("test.com")
    expect(scan?.status).toBe("failed")
    expect(scan?.failure).toMatchObject({
      kind: "container_exit_failed",
      message: expect.stringContaining("Engine Container exited with code 1"),
    })
    expect(failedTask).toMatchObject({
      stepId: "subdomain_discovery",
      failureKind: "container_exit_failed",
      error: expect.stringContaining("Engine Container exited with code 1"),
    })
    expect(logs.results.some((log) => log.content.includes("recon is required"))).toBe(true)
  })

  it("supports scan history BusinessListQuery filters and createdAt ordering in mock data", () => {
    const filtered = getMockScans({
      pageSize: 10,
      filter: '(status=="running" || status=="failed") && targetName="acme"',
      orderBy: "createdAt desc",
    })

    expect(filtered.results.map((scan) => scan.target?.name)).toEqual(["api.acme.com", "acme.io"])
    expect(filtered.results[0]?.status).toBe("running")

    const ascending = getMockScans({ pageSize: 2, orderBy: "createdAt" })
    const descending = getMockScans({ pageSize: 2, orderBy: "createdAt desc" })

    expect(new Date(ascending.results[0]?.createdAt ?? 0).getTime()).toBeLessThanOrEqual(
      new Date(ascending.results[1]?.createdAt ?? 0).getTime()
    )
    expect(new Date(descending.results[0]?.createdAt ?? 0).getTime()).toBeGreaterThanOrEqual(
      new Date(descending.results[1]?.createdAt ?? 0).getTime()
    )
    expect(descending.totalSize).toBe(descending.total)
  })
})
