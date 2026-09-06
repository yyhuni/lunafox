import { act, renderHook, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { useQuickScanDialogState } from "@/components/scan/quick-scan-dialog-state"
import type { ScanWorkflow } from "@/types/scan-workflow.types"

const hookMocks = vi.hoisted(() => ({
  workflows: [] as ScanWorkflow[],
  loadWorkflowProfile: vi.fn(),
  quickScan: { isPending: false, mutateAsync: vi.fn() },
}))

vi.mock("@/hooks/use-scan-workflows", () => ({
  useScanWorkflows: () => ({ data: hookMocks.workflows, isLoading: false, isError: false }),
  useLoadScanWorkflowProfile: () => hookMocks.loadWorkflowProfile,
}))
vi.mock("@/hooks/use-scans", () => ({ useQuickScan: () => hookMocks.quickScan }))
vi.mock("@/hooks/use-engine-catalog", () => ({
  useEngineCatalogDetails: () => ({ data: [], isLoading: false, isError: false, errors: [] }),
}))

describe("useQuickScanDialogState", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    hookMocks.workflows = [createWorkflow("full", true), createWorkflow("unavailable", false)]
    hookMocks.loadWorkflowProfile.mockResolvedValue({
      name: "scanWorkflows/full/profile",
      scanWorkflow: "scanWorkflows/full",
      configuration: { steps: { step: { enabled: false, engineConfig: { recon: { enabled: true, timeout: 60 } } } } },
    })
  })

  it("打开后选择第一个可执行工作流并加载其单例 Profile", async () => {
    const { result } = renderHook(() => useQuickScanDialogState({ t: (key) => key }))
    act(() => result.current.handleClose(true))

    await waitFor(() => {
      expect(result.current.selectedWorkflowNames).toEqual(["full"])
      expect(result.current.configuration).toContain("step")
    })
    expect(hookMocks.loadWorkflowProfile).toHaveBeenCalledWith("full")
  })

  it("新建快速扫描默认使用扫描快照，且允许显式切换来源", () => {
    const { result } = renderHook(() => useQuickScanDialogState({ t: (key) => key }))

    expect(result.current.inputSource).toBe("scanSnapshot")

    act(() => result.current.setInputSource("targetInventory"))
    expect(result.current.inputSource).toBe("targetInventory")

    act(() => result.current.handleClose(false))
    expect(result.current.inputSource).toBe("scanSnapshot")
  })
})

function createWorkflow(name: string, isExecutable: boolean): ScanWorkflow {
  const step = { stageId: "stage", stepId: "step", engineId: "engine.lunafox.subdomain_discovery" }
  const workflowStep = { ...step, profileDefaultEnabled: false }
  return { name, displayName: name, description: "", stages: [{ stageId: "stage", steps: [workflowStep] }], steps: [workflowStep], isBuiltin: false, isExecutable, etag: "etag", createTime: "", updateTime: "" }
}
