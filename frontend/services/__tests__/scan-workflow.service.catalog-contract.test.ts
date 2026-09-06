import { beforeEach, describe, expect, it, vi } from "vitest"

const apiClientMocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), patch: vi.fn() }))
vi.mock("@/lib/api-client", () => ({ default: apiClientMocks }))

import { getScanWorkflow, getScanWorkflowProfile, listScanWorkflows } from "@/services/scan-workflow.service"

const workflow = {
  name: "scanWorkflows/default",
  displayName: "Default Scan",
  description: "Run the default scan.",
  isBuiltin: true,
  isExecutable: true,
  etag: "etag-1",
  createTime: "2026-07-29T00:00:00Z",
  updateTime: "2026-07-29T00:00:00Z",
  stages: [{ stageId: "discovery", steps: [{ stepId: "discover", engineId: "engine.lunafox.discovery", profileDefaultEnabled: true }]}],
}

describe("scan-workflow.service catalog contract", () => {
  beforeEach(() => vi.clearAllMocks())

  it("使用分页查询并只保留纯编排步骤", async () => {
    apiClientMocks.get.mockResolvedValue({ data: { results: [workflow], nextPageToken: "next", totalSize: 2 } })
    const result = await listScanWorkflows({ pageSize: 20, pageToken: "token", filter: "default" })
    expect(apiClientMocks.get).toHaveBeenCalledWith("/scanWorkflows", { params: { pageSize: 20, pageToken: "token", filter: "default" } })
    expect(result).toMatchObject({ nextPageToken: "next", totalSize: 2 })
    expect(result.scanWorkflows[0]).toMatchObject({ name: "scanWorkflows/default", isBuiltin: true })
    expect(result.scanWorkflows[0]?.stages?.[0]?.steps[0]).not.toHaveProperty("engineConfig")
    expect(result.scanWorkflows[0]?.stages?.[0]?.steps[0]?.profileDefaultEnabled).toBe(true)
  })

  it("按工作流资源读取详情和单例 Profile", async () => {
    apiClientMocks.get.mockResolvedValueOnce({ data: workflow }).mockResolvedValueOnce({
      data: {
        name: "scanWorkflows/default/profile",
        scanWorkflow: "scanWorkflows/default",
        configuration: { steps: { discover: { enabled: false, engineConfig: { recon: { enabled: true } } } } },
      },
    })
    await expect(getScanWorkflow("default")).resolves.toMatchObject({ name: "scanWorkflows/default" })
    await expect(getScanWorkflowProfile("default")).resolves.toMatchObject({ scanWorkflow: "scanWorkflows/default" })
    expect(apiClientMocks.get).toHaveBeenNthCalledWith(1, "/scanWorkflows/default")
    expect(apiClientMocks.get).toHaveBeenNthCalledWith(2, "/scanWorkflows/default/profile")
  })

  it("拒绝缺失或非布尔的 Workflow Step Profile 默认值", async () => {
    apiClientMocks.get.mockResolvedValue({
      data: {
        ...workflow,
        stages: [{ stageId: "discovery", steps: [{ stepId: "discover", engineId: "engine.lunafox.discovery" }] }],
      },
    })
    await expect(getScanWorkflow("default")).rejects.toThrow(/pure orchestration topology/)
  })
})
