import { beforeEach, describe, expect, it, vi } from "vitest"

const apiMocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  delete: vi.fn(),
}))

vi.mock("@/lib/api-client", () => ({
  api: apiMocks,
}))

import { api } from "@/lib/api-client"
import {
  batchStopScans,
  bulkInitiateScan,
  getScan,
  getScans,
  getScanStatistics,
  initiateScan,
  quickScan,
} from "@/services/scan.service"

const validSubdomainConfiguration = {
  steps: {
    subdomain_discovery: {
      enabled: true,
      engineConfig: { recon: { enabled: true, timeout: 3600 } },
    },
  },
}

describe("scan.service contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("creates a target scan through canonical batchCreate with one request item", async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: {
        count: 1,
        createdCount: 1,
        skipped: [],
        failed: [],
        scans: [
          {
            id: 11,
            name: "scans/11",
            targetId: 7,
            scanWorkflow: "scanWorkflows/subdomain_discovery",
            plannedEngineIds: [],
            triggerType: "manual",
            inputSource: "targetInventory",
            status: "pending",
            progress: 0,
            currentStage: "",
            createdAt: "2026-06-14T01:02:03Z",
          },
        ],
      },
    } as never)

    const result = await initiateScan({
      targetId: 7,
      scanWorkflow: "subdomain_discovery",
      inputSource: "targetInventory",
      configuration: validSubdomainConfiguration,
    })

    expect(api.post).toHaveBeenCalledTimes(1)
    expect(api.post).toHaveBeenCalledWith("/scans:batchCreate", {
      requests: [{ target: "targets/7" }],
      scanWorkflow: "scanWorkflows/subdomain_discovery",
      inputSource: "targetInventory",
      configuration: validSubdomainConfiguration,
    })
    expect(result.scans).toEqual([
      {
        id: 11,
        target: 7,
        scanWorkflow: "subdomain_discovery",
        status: "pending",
        inputSource: "targetInventory",
        createdAt: "2026-06-14T01:02:03Z",
        updatedAt: "2026-06-14T01:02:03Z",
      },
    ])
  })

  it("bulk scans use one canonical batchCreate request for target and organization scopes", async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: {
        count: 2,
        createdCount: 2,
        skipped: [],
        failed: [],
        scans: [
          {
            id: 11,
            name: "scans/11",
            targetId: 7,
            scanWorkflow: "scanWorkflows/subdomain_discovery",
            plannedEngineIds: [],
            triggerType: "manual",
            inputSource: "scanSnapshot",
            status: "pending",
            progress: 0,
            currentStage: "",
            createdAt: "2026-06-14T01:02:03Z",
          },
          {
            id: 12,
            name: "scans/12",
            targetId: 8,
            scanWorkflow: "scanWorkflows/subdomain_discovery",
            plannedEngineIds: [],
            triggerType: "manual",
            inputSource: "scanSnapshot",
            status: "pending",
            progress: 0,
            currentStage: "",
            createdAt: "2026-06-14T01:03:03Z",
          },
        ],
      },
    } as never)

    const result = await bulkInitiateScan({
      targetIds: [7, 8],
      organizationIds: [5],
      scanWorkflow: "subdomain_discovery",
      inputSource: "scanSnapshot",
      configuration: validSubdomainConfiguration,
    })

    expect(api.post).toHaveBeenCalledTimes(1)
    expect(api.post).toHaveBeenCalledWith("/scans:batchCreate", {
      requests: [
        { target: "targets/7" },
        { target: "targets/8" },
        { organization: "organizations/5" },
      ],
      scanWorkflow: "scanWorkflows/subdomain_discovery",
      inputSource: "scanSnapshot",
      configuration: validSubdomainConfiguration,
    })
    expect(result).toMatchObject({
      message: "Scan initiated successfully",
      count: 2,
    })
  })

  it("quick scans use the backend quickCreate custom method with target names and one workflow resource", async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: {
        count: 1,
        targetStats: { created: 1, skipped: 0, failed: 0 },
        assetStats: { websites: 0, endpoints: 0 },
        errors: [],
        scans: [
          {
            id: 41,
            name: "scans/41",
            targetId: 7,
            scanWorkflow: "scanWorkflows/subdomain_discovery",
            plannedEngineIds: [],
            triggerType: "manual",
            inputSource: "targetInventory",
            status: "pending",
            progress: 0,
            currentStage: "",
            createdAt: "2026-06-14T01:02:03Z",
          },
        ],
      },
    } as never)

    const result = await quickScan({
      targets: [{ name: "example.com" }, { name: "api.example.com" }],
      scanWorkflow: "subdomain_discovery",
      inputSource: "targetInventory",
      configuration: validSubdomainConfiguration,
    })

    expect(api.post).toHaveBeenCalledWith("/scans:quickCreate", {
      targets: ["example.com", "api.example.com"],
      scanWorkflow: "scanWorkflows/subdomain_discovery",
      inputSource: "targetInventory",
      configuration: validSubdomainConfiguration,
    })
    expect(result.count).toBe(1)
    expect(result.scans[0]).toMatchObject({ id: 41, target: 7, status: "pending", inputSource: "targetInventory" })
  })

  it("quick scans fail fast before the network when scanWorkflow is missing", async () => {
    await expect(
      quickScan({
        targets: [{ name: "example.com" }],
        configuration: { steps: {} },
      } as unknown as Parameters<typeof quickScan>[0])
    ).rejects.toThrow("scanWorkflow is required")

    expect(api.post).not.toHaveBeenCalled()
  })

  it.each([
    ["missing", undefined],
    ["empty", ""],
    ["unknown", "legacy"],
  ])("quick scans reject a %s inputSource before the network", async (_label, inputSource) => {
    await expect(
      quickScan({
        targets: [{ name: "example.com" }],
        scanWorkflow: "subdomain_discovery",
        configuration: validSubdomainConfiguration,
        inputSource,
      } as unknown as Parameters<typeof quickScan>[0])
    ).rejects.toThrow(/inputSource/)

    expect(api.post).not.toHaveBeenCalled()
  })

  it("rejects a list response that omits the required triggerType", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: { results: [{ plannedEngineIds: [] }], total: 1, page: 1, pageSize: 10, totalPages: 1 },
    } as never)

    await expect(getScans()).rejects.toThrow("Scan response is missing triggerType")
  })

  it("rejects a detail response with an unknown triggerType", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: { triggerType: "legacy", plannedEngineIds: [] },
    } as never)

    await expect(getScan(91)).rejects.toThrow("Scan response has an invalid triggerType")
  })

  it("rejects a list response that omits the required inputSource", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [{ triggerType: "manual", plannedEngineIds: [] }],
        total: 1,
        page: 1,
        pageSize: 10,
        totalPages: 1,
      },
    } as never)

    await expect(getScans()).rejects.toThrow("Scan response is missing inputSource")
  })

  it("rejects a detail response with an unknown inputSource", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: { triggerType: "manual", inputSource: "legacy", plannedEngineIds: [] },
    } as never)

    await expect(getScan(91)).rejects.toThrow("Scan response has an invalid inputSource")
  })

  it("rejects a batch-create response that omits the required triggerType", async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: { count: 1, createdCount: 1, skipped: [], failed: [], scans: [{ plannedEngineIds: [] }] },
    } as never)

    await expect(bulkInitiateScan({ targetIds: [7], scanWorkflow: "subdomain_discovery", inputSource: "scanSnapshot", configuration: validSubdomainConfiguration }))
      .rejects.toThrow("Scan response is missing triggerType")
  })

  it("rejects a quick-create response with an unknown triggerType", async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: {
        count: 1,
        targetStats: { created: 1, skipped: 0, failed: 0 },
        assetStats: { websites: 0, endpoints: 0 },
        errors: [],
        scans: [{ triggerType: "legacy", plannedEngineIds: [] }],
      },
    } as never)

    await expect(quickScan({ targets: [{ name: "example.com" }], scanWorkflow: "subdomain_discovery", inputSource: "scanSnapshot", configuration: validSubdomainConfiguration }))
      .rejects.toThrow("Scan response has an invalid triggerType")
  })

  it("bulk organization scans use the canonical batchCreate organization scope", async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: { count: 0, createdCount: 0, skipped: [], failed: [], scans: [] },
    } as never)

    await bulkInitiateScan({
      organizationIds: [3],
      scanWorkflow: "subdomain_discovery",
      inputSource: "scanSnapshot",
      configuration: validSubdomainConfiguration,
    })

    expect(api.post).toHaveBeenCalledWith("/scans:batchCreate", {
      requests: [{ organization: "organizations/3" }],
      scanWorkflow: "scanWorkflows/subdomain_discovery",
      inputSource: "scanSnapshot",
      configuration: validSubdomainConfiguration,
    })
  })

  it("reads scan history statistics from the backend-owned API shape", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        total: 6,
        pending: 1,
        running: 2,
        completed: 1,
        failed: 1,
        cancelled: 1,
        totalVulns: 4,
        totalSubdomains: 5,
        totalEndpoints: 6,
        totalWebsites: 7,
        totalAssets: 18,
        retentionPolicy: {
          minimumRetentionSeconds: 14 * 24 * 60 * 60,
          automaticCleanupEnabled: true,
        },
      },
    } as never)

    await expect(getScanStatistics()).resolves.toMatchObject({
      pending: 1,
      running: 2,
      succeeded: 1,
      failed: 1,
      cancelled: 1,
      retentionPolicy: {
        minimumRetentionSeconds: 14 * 24 * 60 * 60,
        automaticCleanupEnabled: true,
      },
    })
    expect(api.get).toHaveBeenCalledWith("/scanStatistics")
  })

  it("stops selected scans through the bounded canonical batch custom method", async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: {
        stoppedCount: 2,
        skippedCount: 1,
        revokedTaskCount: 3,
      },
    } as never)

    await expect(batchStopScans([7, 8, 9])).resolves.toEqual({
      stoppedCount: 2,
      skippedCount: 1,
      revokedTaskCount: 3,
    })
    expect(api.post).toHaveBeenCalledWith("/scans:batchStop", {
      names: ["scans/7", "scans/8", "scans/9"],
    })
  })

  it.each([
    ["empty", []],
    ["duplicate", [7, 7]],
    ["zero", [0]],
    ["over limit", Array.from({ length: 101 }, (_, index) => index + 1)],
  ])("rejects a %s batch stop before the network", async (_label, ids) => {
    await expect(batchStopScans(ids)).rejects.toThrow(/batchStop/)
    expect(api.post).not.toHaveBeenCalled()
  })

  it("rejects a batch-stop response with a missing or invalid count", async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: { stoppedCount: 1, skippedCount: -1, revokedTaskCount: 0 },
    } as never)

    await expect(batchStopScans([7])).rejects.toThrow("Batch stop response has an invalid skippedCount")
  })

  it("keeps scanWorkflow separate from executed engine IDs", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [
          {
            id: 21,
            name: "scans/21",
            targetId: 7,
            scanWorkflow: "scanWorkflows/full_recon",
            plannedEngineIds: [
              "engine.lunafox.subdomain_discovery",
              "engine.lunafox.port_scan",
              "engine.lunafox.nuclei_vulnerability",
            ],
            triggerType: "manual",
            inputSource: "scanSnapshot",
            status: "running",
            progress: 40,
            currentStage: "port_scan",
            createdAt: "2026-06-14T01:02:03Z",
          },
        ],
        total: 1,
        page: 1,
        pageSize: 10,
        totalPages: 1,
      },
    } as never)

    const result = await getScans({ page: 1, pageSize: 10 })

    expect(result.results[0]?.scanWorkflow).toBe("full_recon")
    expect(result.results[0]).toMatchObject({
      plannedEngineIds: [
        "engine.lunafox.subdomain_discovery",
        "engine.lunafox.port_scan",
        "engine.lunafox.nuclei_vulnerability",
      ],
    })
    expect(result.results[0]).not.toHaveProperty("engineNames")
  })

  it("does not copy scanWorkflow into plannedEngineIds when the list summary is empty", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [
          {
            id: 22,
            name: "scans/22",
            targetId: 7,
            scanWorkflow: "scanWorkflows/full_recon",
            plannedEngineIds: [],
            workflowNames: ["legacy_step_should_not_win"],
            triggerType: "manual",
            inputSource: "scanSnapshot",
            status: "pending",
            progress: 0,
            currentStage: "",
            createdAt: "2026-06-14T01:02:03Z",
          },
        ],
        total: 1,
        page: 1,
        pageSize: 10,
        totalPages: 1,
      },
    } as never)

    const result = await getScans({ page: 1, pageSize: 10 })

    expect(result.results[0]?.scanWorkflow).toBe("full_recon")
    expect(result.results[0]).toMatchObject({ plannedEngineIds: [] })
    expect(result.results[0]).not.toHaveProperty("engineNames")
    expect(result.results[0]).not.toHaveProperty("workflowNames")
  })

  it("passes canonical scan list query controls without merging search into filter", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [],
        totalSize: 0,
        nextPageToken: "",
        pageSize: 10,
      },
    } as never)

    await getScans({
      pageSize: 10,
      pageToken: "cGFnZToy",
      target: 7,
      filter: '(status=="running" || status=="failed") && targetName="acme"',
      orderBy: "createdAt desc",
    })

    expect(api.get).toHaveBeenCalledWith("/scans", {
      params: {
        pageSize: 10,
        pageToken: "cGFnZToy",
        target: 7,
        filter: '(status=="running" || status=="failed") && targetName="acme"',
        orderBy: "createdAt desc",
      },
    })
  })

  it("normalizes AIP pagination fields for scan list callers", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [],
        totalSize: 23,
        nextPageToken: "cGFnZToy",
      },
    } as never)

    const result = await getScans({ pageSize: 10 })

    expect(result.totalSize).toBe(23)
    expect(result.nextPageToken).toBe("cGFnZToy")
    expect(result.total).toBe(23)
    expect(result.page).toBe(1)
    expect(result.pageSize).toBe(10)
    expect(result.totalPages).toBe(3)
  })

  it("maps the authoritative Scan detail Agent projection without a collection lookup", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        id: 91,
        name: "scans/91",
        targetId: 7,
        scanWorkflow: "scanWorkflows/default",
        plannedEngineIds: ["engine.lunafox.subdomain_discovery"],
        triggerType: "scheduled",
        inputSource: "targetInventory",
        status: "running",
        progress: 50,
        currentStage: "discovery",
        createdAt: "2026-08-04T00:00:00Z",
        agentId: 1001,
        agent: "agents/1001",
        agentName: "renamed-agent",
        agentStatus: "online",
        agentHealthState: "paused",
        agentDeleted: false,
        assignmentMode: "pinned",
      },
    } as never)

    const result = await getScan(91)

    expect(api.get).toHaveBeenCalledWith("/scans/91")
    expect(result).toMatchObject({
      agentId: 1001,
      agent: "agents/1001",
      agentName: "renamed-agent",
      agentStatus: "online",
      agentHealthState: "paused",
      agentDeleted: false,
      assignmentMode: "pinned",
    })
  })

  it("rejects a Scan detail whose canonical Agent identity and numeric projection disagree", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        id: 91,
        name: "scans/91",
        targetId: 7,
        scanWorkflow: "scanWorkflows/default",
        plannedEngineIds: [],
        triggerType: "manual",
        inputSource: "scanSnapshot",
        status: "running",
        progress: 50,
        currentStage: "discovery",
        createdAt: "2026-08-04T00:00:00Z",
        agentId: 1002,
        agent: "agents/1001",
        agentName: "agent",
        agentDeleted: false,
        assignmentMode: "automatic",
      },
    } as never)

    await expect(getScan(91)).rejects.toThrow(
      "Scan detail response Agent identity does not match its resource name",
    )
  })

  it("preserves the Server-defined terminal diagnostic snapshot without recalculating delivery state", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        id: 91,
        name: "scans/91",
        targetId: 7,
        scanWorkflow: "scanWorkflows/default",
        plannedEngineIds: ["engine.lunafox.port_scan"],
        triggerType: "manual",
        inputSource: "scanSnapshot",
        status: "failed",
        progress: 100,
        createdAt: "2026-08-04T00:00:00Z",
        assignmentMode: "automatic",
        runtimeTasks: [
          {
            id: 7001,
            name: "scans/91/tasks/7001",
            stepId: "port_scan",
            stageId: "discovery",
            engineId: "engine.lunafox.port_scan",
            status: "failed",
            order: 0,
            diagnostics: {
              compatibilityRevision: "engine-execution-diagnostics-r1",
              availability: "available",
              resultState: "partial",
              failedStage: "result_submit",
              errorType: "result_submit_failed",
              resultTypeWatermarks: [
                {
                  resultType: "network.port",
                  receivedItems: 4,
                  encodedItems: 4,
                  submittedItems: 4,
                  acknowledgedItems: 2,
                  submittedBatches: 2,
                  acknowledgedBatches: 1,
                },
              ],
            },
          },
        ],
      },
    } as never)

    const result = await getScan(91)

    expect(result.runtimeTasks?.[0]?.diagnostics).toEqual({
      compatibilityRevision: "engine-execution-diagnostics-r1",
      availability: "available",
      resultState: "partial",
      failedStage: "result_submit",
      errorType: "result_submit_failed",
      resultTypeWatermarks: [
        {
          resultType: "network.port",
          receivedItems: 4,
          encodedItems: 4,
          submittedItems: 4,
          acknowledgedItems: 2,
          submittedBatches: 2,
          acknowledgedBatches: 1,
        },
      ],
    })
  })

  it("rejects malformed terminal diagnostics before they can be rendered", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        id: 91,
        name: "scans/91",
        targetId: 7,
        scanWorkflow: "scanWorkflows/default",
        plannedEngineIds: ["engine.lunafox.port_scan"],
        triggerType: "manual",
        inputSource: "scanSnapshot",
        status: "failed",
        progress: 100,
        createdAt: "2026-08-04T00:00:00Z",
        assignmentMode: "automatic",
        runtimeTasks: [
          {
            id: 7001,
            name: "scans/91/tasks/7001",
            stepId: "port_scan",
            stageId: "discovery",
            engineId: "engine.lunafox.port_scan",
            status: "failed",
            order: 0,
            diagnostics: {
              availability: "unavailable",
              resultState: "unknown",
              rawError: "must never cross the boundary",
            },
          },
        ],
      },
    } as never)

    await expect(getScan(91)).rejects.toThrow("Scan detail response diagnostics has an unsupported field: rawError")
  })
})
