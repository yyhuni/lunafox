import { beforeEach, describe, expect, it, vi } from "vitest"

const apiMocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
}))

vi.mock("@/lib/api-client", () => ({
  api: apiMocks,
}))

import { api } from "@/lib/api-client"
import {
	batchDeleteScheduledScans,
	batchUpdateScheduledScanStatus,
  createScheduledScan,
  getScheduledScan,
  getScheduledScans,
	getScheduledScanOverviewSummary,
  updateScheduledScan,
} from "@/services/scheduled-scan.service"

const validSubdomainConfiguration = {
  steps: {
    subdomain_discovery: {
      enabled: true,
      engineConfig: { recon: { enabled: true, timeout: 3600 } },
    },
  },
}

function scheduledScanTransport(overrides: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    id: 4,
    name: "scheduledScans/4",
    displayName: "daily",
    scanWorkflow: "scanWorkflows/subdomain_discovery",
    target: "targets/7",
    targetId: 7,
    targetName: "example.com",
    organizationId: null,
    organizationName: null,
    scanMode: "target",
    inputSource: "scanSnapshot",
    cronExpression: "0 2 * * *",
    isEnabled: true,
    nextRunTime: "2026-06-15T00:00:00Z",
    lastRunTime: null,
    runCount: 0,
    successfulHandoffCount: 0,
    failedHandoffCount: 0,
    createdAt: "2026-06-14T01:02:03Z",
    updatedAt: "2026-06-14T01:02:03Z",
    ...overrides,
  }
}

function scheduledScanOverviewTransport(overrides: Record<string, unknown> = {}): Record<string, unknown> {
	return {
		asOfTime: "2026-06-14T08:00:00Z",
		enabledScheduledScanCount: 3,
		pausedScheduledScanCount: 1,
		todayScheduledScanCount: 2,
		next24HoursScheduledScanCount: 3,
		upcomingScheduledScans: [{
			name: "scheduledScans/4",
			displayName: "daily",
			organization: null,
			organizationDisplayName: null,
			target: "targets/7",
			targetDisplayName: "example.com",
			nextRunTime: "2026-06-14T09:00:00Z",
		}],
		...overrides,
	}
}

describe("scheduled-scan.service contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("列表通过 AIP 集合路径查询并只发送 camelCase 参数", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        scheduledScans: [
          {
            id: 4,
            name: "scheduledScans/4",
            displayName: "daily",
            scanWorkflow: "scanWorkflows/subdomain_discovery",
            target: "targets/7",
            targetId: 7,
            targetName: "example.com",
            organizationId: null,
            organizationName: null,
            scanMode: "target",
            inputSource: "scanSnapshot",
            cronExpression: "0 2 * * *",
            isEnabled: true,
            nextRunTime: "2026-06-15T00:00:00Z",
            lastRunTime: null,
            runCount: 0,
            successfulHandoffCount: 0,
            failedHandoffCount: 0,
            createdAt: "2026-06-14T01:02:03Z",
            updatedAt: "2026-06-14T01:02:03Z",
          },
        ],
        totalSize: 1,
      },
    } as never)

    const result = await getScheduledScans({
      pageSize: 25,
      pageToken: "scheduled-scan-cursor-2",
      search: "demo",
      targetId: 7,
      organizationId: 9,
    })

    expect(api.get).toHaveBeenCalledWith("/scheduledScans", {
      params: {
        pageSize: 25,
        pageToken: "scheduled-scan-cursor-2",
        filter: "demo",
        targetId: 7,
        organizationId: 9,
      },
    })
    expect(vi.mocked(api.get).mock.calls[0]?.[1]?.params).not.toHaveProperty("target_id")
    expect(vi.mocked(api.get).mock.calls[0]?.[1]?.params).not.toHaveProperty("organization_id")
    expect(result.scheduledScans[0]).toMatchObject({
      id: 4,
      name: "daily",
      displayName: "daily",
      resourceName: "scheduledScans/4",
      scanWorkflow: "subdomain_discovery",
      inputSource: "scanSnapshot",
    })
  })

  it("首屏不发送 pageToken，并保留响应的 nextPageToken", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        scheduledScans: [],
        totalSize: 25,
        nextPageToken: "scheduled-scan-cursor-2",
      },
    } as never)

    const result = await getScheduledScans({ pageSize: 25 })

    expect(api.get).toHaveBeenCalledWith("/scheduledScans", {
      params: { pageSize: 25 },
    })
    expect(result.nextPageToken).toBe("scheduled-scan-cursor-2")
  })

	it("概览使用独立 custom method、固定空查询并投影紧凑时间线字段", async () => {
		vi.mocked(api.get).mockResolvedValue({ data: scheduledScanOverviewTransport() } as never)

		const result = await getScheduledScanOverviewSummary()

		expect(api.get).toHaveBeenCalledWith("/scheduledScans:summarize", {
			params: {},
		})
		expect(result).toMatchObject({
			enabledScheduledScanCount: 3,
			pausedScheduledScanCount: 1,
			upcomingScheduledScans: [{
				id: 4,
				resourceName: "scheduledScans/4",
				displayName: "daily",
				scanMode: "target",
				targetName: "example.com",
			}],
		})
	})

	it.each([
		["negative count", scheduledScanOverviewTransport({ enabledScheduledScanCount: -1 }), "enabledScheduledScanCount"],
		["too many upcoming tasks", scheduledScanOverviewTransport({ upcomingScheduledScans: Array(6).fill({}) }), "upcomingScheduledScans"],
		["missing scoped display name", scheduledScanOverviewTransport({ upcomingScheduledScans: [{ name: "scheduledScans/4", displayName: "daily", organization: null, organizationDisplayName: null, target: "targets/7", targetDisplayName: null, nextRunTime: "2026-06-14T09:00:00Z" }] }), "upcomingScheduledScans[0].targetDisplayName"],
	])("rejects overview response with %s", async (_name, payload, expectedField) => {
		vi.mocked(api.get).mockResolvedValue({ data: payload } as never)
		await expect(getScheduledScanOverviewSummary()).rejects.toThrow(`invalid ${expectedField}`)
	})

  it.each([
    ["missing successful total", "successfulHandoffCount", undefined, "successfulHandoffCount"],
    ["negative failed total", "failedHandoffCount", -1, "failedHandoffCount"],
    ["fractional successful total", "successfulHandoffCount", 1.5, "successfulHandoffCount"],
    ["string failed total", "failedHandoffCount", "1", "failedHandoffCount"],
  ])("rejects a %s in the Scheduled Scan outcome projection", async (_name, field, value, expectedField) => {
    const scan = scheduledScanTransport()
    if (value === undefined) {
      delete scan[field]
    } else {
      scan[field] = value
    }
    vi.mocked(api.get).mockResolvedValue({ data: { scheduledScans: [scan] } } as never)

    await expect(getScheduledScans()).rejects.toThrow(`invalid ${expectedField}`)
  })

  it("rejects an invalid outcome projection from Scheduled Scan Get", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: scheduledScanTransport({ failedHandoffCount: -1 }),
    } as never)

    await expect(getScheduledScan(4)).rejects.toThrow("invalid failedHandoffCount")
  })

  it.each([
    ["missing", undefined],
    ["unknown", "legacy"],
  ])("rejects a %s inputSource in Scheduled Scan responses", async (_label, inputSource) => {
    vi.mocked(api.get).mockResolvedValue({
      data: { scheduledScans: [scheduledScanTransport({ inputSource })] },
    } as never)

    await expect(getScheduledScans()).rejects.toThrow(/inputSource/)
  })

  it("创建计划扫描时将前端 id 转成 AIP resource name", async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: {
        id: 4,
        name: "scheduledScans/4",
        displayName: "daily",
        scanWorkflow: "scanWorkflows/subdomain_discovery",
        target: "targets/7",
        targetId: 7,
        targetName: "example.com",
        scanMode: "target",
        inputSource: "targetInventory",
        cronExpression: "0 2 * * *",
        isEnabled: true,
        nextRunTime: "2026-06-15T00:00:00Z",
        lastRunTime: null,
        runCount: 0,
        successfulHandoffCount: 0,
        failedHandoffCount: 0,
        createdAt: "2026-06-14T01:02:03Z",
        updatedAt: "2026-06-14T01:02:03Z",
      },
    } as never)

    await createScheduledScan({
      displayName: "daily",
      scanWorkflow: "subdomain_discovery",
      configuration: validSubdomainConfiguration,
      targetId: 7,
      agentId: 42,
      cronExpression: "0 2 * * *",
      isEnabled: true,
      inputSource: "targetInventory",
    })

    expect(api.post).toHaveBeenCalledWith("/scheduledScans", {
      displayName: "daily",
      scanWorkflow: "scanWorkflows/subdomain_discovery",
      inputSource: "targetInventory",
      configuration: validSubdomainConfiguration,
      target: "targets/7",
      agent: "agents/42",
      cronExpression: "0 2 * * *",
      isEnabled: true,
    })
  })

  it("创建计划扫描缺失来源时在网络调用前失败", async () => {
    await expect(
      createScheduledScan({
        displayName: "daily",
        scanWorkflow: "subdomain_discovery",
        configuration: validSubdomainConfiguration,
        targetId: 7,
        cronExpression: "0 2 * * *",
        inputSource: undefined,
      } as unknown as Parameters<typeof createScheduledScan>[0])
    ).rejects.toThrow("Scheduled scan request is missing inputSource")

    expect(api.post).not.toHaveBeenCalled()
  })

  it("更新计划扫描可清空或替换未来触发的指定节点", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: { id: 4, name: "scheduledScans/4", displayName: "daily", scanWorkflow: "scanWorkflows/subdomain_discovery", organizationId: null, organizationName: null, targetId: 7, targetName: "example.com", scanMode: "target", inputSource: "scanSnapshot", cronExpression: "0 2 * * *", isEnabled: true, nextRunTime: "2026-06-15T02:00:00Z", lastRunTime: null, runCount: 0, successfulHandoffCount: 0, failedHandoffCount: 0, createdAt: "2026-06-14T01:02:03Z", updatedAt: "2026-06-14T01:02:03Z" } } as never)

    await updateScheduledScan(4, { agentId: 43 })
    expect(api.patch).toHaveBeenLastCalledWith("/scheduledScans/4", { name: "scheduledScans/4", agent: "agents/43", updateMask: "agent" })

    await updateScheduledScan(4, { agentId: null })
    expect(api.patch).toHaveBeenLastCalledWith("/scheduledScans/4", { name: "scheduledScans/4", agent: "", updateMask: "agent" })

  })

  it("仅在来源变更时把 inputSource 加入更新请求和 updateMask", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: scheduledScanTransport({ inputSource: "targetInventory" }) } as never)

    await updateScheduledScan(4, { inputSource: "targetInventory" })

    expect(api.patch).toHaveBeenCalledWith("/scheduledScans/4", {
      name: "scheduledScans/4",
      inputSource: "targetInventory",
      updateMask: "inputSource",
    })
  })

  it("更换计划扫描工作流时与完整配置使用同一个 updateMask", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: scheduledScanTransport() } as never)

    await updateScheduledScan(4, {
      scanWorkflow: "subdomain_discovery",
      configuration: validSubdomainConfiguration,
    })

    expect(api.patch).toHaveBeenCalledWith("/scheduledScans/4", {
      name: "scheduledScans/4",
      scanWorkflow: "scanWorkflows/subdomain_discovery",
      configuration: validSubdomainConfiguration,
      updateMask: "scanWorkflow,configuration",
    })
  })

	it("批量删除复用 AIP 定时扫描单项删除端点", async () => {
    vi.mocked(api.delete).mockResolvedValue({ data: { message: "ok" } } as never)

    const result = await batchDeleteScheduledScans([3, 5])

    expect(api.delete).toHaveBeenCalledWith("/scheduledScans/3")
    expect(api.delete).toHaveBeenCalledWith("/scheduledScans/5")
    expect(result).toEqual({
      message: "ok",
      deletedCount: 2,
	    })
	})

	it("批量状态更新发送 canonical names 和精确 updateMask", async () => {
		vi.mocked(api.post).mockResolvedValue({ data: { updatedCount: 2 } } as never)

		const result = await batchUpdateScheduledScanStatus({ ids: [3, 5], isEnabled: false })

		expect(api.post).toHaveBeenCalledWith("/scheduledScans:batchUpdate", {
			requests: [
				{ name: "scheduledScans/3", isEnabled: false, updateMask: "isEnabled" },
				{ name: "scheduledScans/5", isEnabled: false, updateMask: "isEnabled" },
			],
		})
		expect(result).toEqual({ updatedCount: 2 })
	})

	it.each([
		["empty", []],
		["too many", Array.from({ length: 101 }, (_, index) => index + 1)],
		["duplicate", [3, 3]],
		["invalid", [0]],
	])("在网络调用前拒绝批量状态更新的 %s ID", async (_label, ids) => {
		await expect(batchUpdateScheduledScanStatus({ ids, isEnabled: true })).rejects.toThrow()
		expect(api.post).not.toHaveBeenCalled()
	})

	it("拒绝与请求数量不一致的批量状态响应", async () => {
		vi.mocked(api.post).mockResolvedValue({ data: { updatedCount: 1 } } as never)

		await expect(
			batchUpdateScheduledScanStatus({ ids: [3, 5], isEnabled: true })
		).rejects.toThrow("unexpected updatedCount")
	})
})
