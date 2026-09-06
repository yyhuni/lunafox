import { act, renderHook, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { useEditScheduledScanDialogState } from "@/components/scan/scheduled/edit-scheduled-scan-dialog-state"
import type { ScheduledScan } from "@/types/scheduled-scan.types"
import type { ScanWorkflow } from "@/types/scan-workflow.types"

const stateMocks = vi.hoisted(() => ({
  updateScheduledScan: vi.fn(),
  loadWorkflowProfile: vi.fn(),
  workflows: [] as ScanWorkflow[],
  engineCatalog: {
    data: [{}] as unknown[] | undefined,
    isLoading: false,
    isError: false,
  },
}))

vi.mock("@/hooks/use-scheduled-scans", () => ({
  useUpdateScheduledScan: () => ({ mutate: stateMocks.updateScheduledScan, isPending: false }),
}))

vi.mock("@/hooks/use-targets", () => ({
  useTargets: () => ({ data: { targets: [] } }),
}))

vi.mock("@/hooks/use-scan-workflows", () => ({
  useLoadScanWorkflowProfile: () => stateMocks.loadWorkflowProfile,
  useScanWorkflows: () => ({ data: stateMocks.workflows, isLoading: false, isError: false }),
}))

vi.mock("@/hooks/use-engine-catalog", () => ({
  useEngineCatalogDetails: () => stateMocks.engineCatalog,
}))

vi.mock("@/lib/engine-catalog", () => ({
  buildWorkflowWithEngineCatalog: () => ({}),
}))

const scheduledScan: ScheduledScan = {
  id: 1,
  name: "Daily scan",
  displayName: "Daily scan",
  scanWorkflow: "default",
  configuration: {
    steps: {
      discovery: { enabled: true, engineConfig: { options: { timeout: 30 } } },
    },
  },
  organizationId: 1,
  organizationName: "Acme",
  targetId: null,
  targetName: null,
  scanMode: "organization",
  inputSource: "scanSnapshot",
  cronExpression: "0 2 * * *",
  isEnabled: true,
  nextRunTime: "2026-08-16T02:00:00Z",
  lastRunTime: null,
  runCount: 0,
  successfulHandoffCount: 0,
  failedHandoffCount: 0,
  createdAt: "2026-08-15T00:00:00Z",
  updatedAt: "2026-08-15T00:00:00Z",
}

function createWorkflow(id: string): ScanWorkflow {
  const step = { stageId: "recon", stepId: "discovery", engineId: "builtin.discovery", profileDefaultEnabled: true }
  return {
    name: `scanWorkflows/${id}`,
    displayName: id,
    description: "",
    stages: [{ stageId: "recon", steps: [step] }],
    steps: [step],
    isBuiltin: false,
    isExecutable: true,
    etag: "etag",
    createTime: "",
    updateTime: "",
  }
}

function renderState() {
  return renderHook(() => useEditScheduledScanDialogState({
    open: true,
    scheduledScan,
    onOpenChange: vi.fn(),
    t: (key) => key,
  }))
}

describe("useEditScheduledScanDialogState", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    stateMocks.workflows = [createWorkflow("default"), createWorkflow("recon")]
    stateMocks.engineCatalog = { data: [{}], isLoading: false, isError: false }
  })

  it("以已保存配置初始化编辑草稿", async () => {
    const { result } = renderState()

    await waitFor(() => {
      expect(result.current.configuration).toContain("discovery")
    })

    expect(result.current.isConfigEdited).toBe(false)
    expect(result.current.inputSource).toBe("scanSnapshot")
  })

  it("编辑时保留已持久化来源，并且只在变化时提交它", async () => {
    const { result } = renderState()

    await waitFor(() => {
      expect(result.current.scanWorkflow).toBe("scanWorkflows/default")
    })

    act(() => result.current.handleSubmit())
    expect(stateMocks.updateScheduledScan).toHaveBeenLastCalledWith(
      {
        id: scheduledScan.id,
        data: expect.not.objectContaining({ inputSource: expect.anything() }),
      },
      expect.any(Object)
    )

    act(() => result.current.setInputSource("targetInventory"))
    act(() => result.current.handleSubmit())
    expect(stateMocks.updateScheduledScan).toHaveBeenLastCalledWith(
      {
        id: scheduledScan.id,
        data: expect.objectContaining({ inputSource: "targetInventory" }),
      },
      expect.any(Object)
    )
  })

  it("用新工作流的 Profile 配置一并更新定时扫描", async () => {
    stateMocks.loadWorkflowProfile.mockResolvedValue({
      name: "scanWorkflows/recon/profile",
      scanWorkflow: "scanWorkflows/recon",
      configuration: {
        steps: {
          discovery: { enabled: true, engineConfig: { options: { timeout: 60 } } },
        },
      },
    })

    const { result } = renderState()

    await waitFor(() => {
      expect(result.current.scanWorkflow).toBe("scanWorkflows/default")
    })

    await act(async () => {
      await result.current.handleWorkflowNamesChange(["scanWorkflows/recon"])
    })

    expect(stateMocks.loadWorkflowProfile).toHaveBeenCalledWith("scanWorkflows/recon")
    expect(result.current.scanWorkflow).toBe("scanWorkflows/recon")

    act(() => result.current.handleSubmit())

    expect(stateMocks.updateScheduledScan).toHaveBeenCalledWith(
      {
        id: scheduledScan.id,
        data: expect.objectContaining({
          scanWorkflow: "scanWorkflows/recon",
          configuration: expect.stringContaining("discovery"),
        }),
      },
      expect.any(Object)
    )
  })

  it("Profile 加载失败时保留原工作流且不提交替换配置", async () => {
    stateMocks.loadWorkflowProfile.mockRejectedValue(new Error("Profile unavailable"))

    const { result } = renderState()

    await waitFor(() => {
      expect(result.current.scanWorkflow).toBe("scanWorkflows/default")
    })

    await act(async () => {
      await result.current.handleWorkflowNamesChange(["scanWorkflows/recon"])
    })

    expect(result.current.scanWorkflow).toBe("scanWorkflows/default")

    act(() => result.current.handleSubmit())

    expect(stateMocks.updateScheduledScan).toHaveBeenCalledWith(
      {
        id: scheduledScan.id,
        data: expect.not.objectContaining({ configuration: expect.anything() }),
      },
      expect.any(Object)
    )
  })

  it("修改配置后在更新请求中提交配置", async () => {
    const { result } = renderState()

    await waitFor(() => {
      expect(result.current.scanWorkflow).toBe("scanWorkflows/default")
    })

    act(() => {
      result.current.handleManualConfigChange([
        "steps:",
        "  discovery:",
        "    enabled: true",
        "    engineConfig:",
        "      options:",
        "        timeout: 60",
      ].join("\n"))
    })
    act(() => result.current.handleSubmit())

    expect(stateMocks.updateScheduledScan).toHaveBeenCalledWith(
      {
        id: scheduledScan.id,
        data: expect.objectContaining({
          configuration: expect.stringContaining("timeout: 60"),
        }),
      },
      expect.any(Object)
    )
  })

  it("引擎目录失败时仍可保存未修改配置的基础字段", async () => {
    stateMocks.engineCatalog = { data: undefined, isLoading: false, isError: true }
    const { result } = renderState()

    await waitFor(() => {
      expect(result.current.scanWorkflow).toBe("scanWorkflows/default")
    })

    act(() => result.current.setDisplayName("Updated daily scan"))
    act(() => result.current.handleSubmit())

    expect(stateMocks.updateScheduledScan).toHaveBeenCalledWith(
      {
        id: scheduledScan.id,
        data: expect.not.objectContaining({ configuration: expect.anything() }),
      },
      expect.any(Object)
    )
  })

  it("引擎目录失败时阻止提交已修改的配置", async () => {
    stateMocks.engineCatalog = { data: undefined, isLoading: false, isError: true }
    const { result } = renderState()

    await waitFor(() => {
      expect(result.current.scanWorkflow).toBe("scanWorkflows/default")
    })

    act(() => result.current.handleManualConfigChange("steps: {}"))
    expect(result.current.isConfigurationSaveBlocked).toBe(true)

    act(() => result.current.handleSubmit())

    expect(stateMocks.updateScheduledScan).not.toHaveBeenCalled()
  })
})
