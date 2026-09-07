import { act, renderHook } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { useScheduledScanDialogState } from "@/components/scan/scheduled/scheduled-scan-dialog-state"

const stateMocks = vi.hoisted(() => ({
  currentStep: 3,
  goToNextStep: vi.fn(),
  goToPrevStep: vi.fn(),
  resetStep: vi.fn(),
  createScheduledScan: vi.fn(),
  loadWorkflowProfile: vi.fn(),
  workflows: [] as Array<Record<string, unknown>>,
  applyProfileConfiguration: vi.fn(),
}))

vi.mock("@/hooks/use-step", () => ({
  useStep: () => [stateMocks.currentStep, {
    goToNextStep: stateMocks.goToNextStep,
    goToPrevStep: stateMocks.goToPrevStep,
    reset: stateMocks.resetStep,
  }],
}))

vi.mock("@/hooks/use-scheduled-scans", () => ({
  useCreateScheduledScan: () => ({
    mutate: stateMocks.createScheduledScan,
    isPending: false,
  }),
}))

vi.mock("@/hooks/use-scan-workflows", () => ({
  useLoadScanWorkflowProfile: () => stateMocks.loadWorkflowProfile,
  useScanWorkflows: () => ({ data: stateMocks.workflows, isLoading: false, isError: false }),
}))

vi.mock("@/hooks/use-engine-catalog", () => ({
  useEngineCatalogDetails: () => ({ data: [], isLoading: false, isError: false }),
}))

vi.mock("@/components/scan/scheduled/scheduled-scan-dialog-state-hooks", () => ({
  useScheduledScanSearch: () => ({
    orgSearchInput: "",
    setOrgSearchInput: vi.fn(),
    orgPage: 1,
    setOrgPage: vi.fn(),
    orgPageSize: 8,
    setOrgPageSize: vi.fn(),
    organizationPaginationInfo: undefined,
    targetSearchInput: "",
    setTargetSearchInput: vi.fn(),
    targetPage: 1,
    setTargetPage: vi.fn(),
    targetPageSize: 10,
    setTargetPageSize: vi.fn(),
    targetPaginationInfo: undefined,
    handleOrgSearch: vi.fn(),
    isOrgFetching: false,
    isTargetFetching: false,
    organizations: [],
    targets: [],
  }),
  useScheduledScanConfigState: () => ({
    configuration: "steps:\n  discovery:\n    enabled: true",
    isConfigEdited: false,
    isYamlValid: true,
    showOverwriteConfirm: false,
    pendingConfigChange: null,
    setShowOverwriteConfirm: vi.fn(),
    setPendingConfigChange: vi.fn(),
    handlePresetConfigChange: vi.fn(),
    applyProfileConfiguration: stateMocks.applyProfileConfiguration,
    handleConfigSync: vi.fn(),
    handleManualConfigChange: vi.fn(),
    handleOverwriteConfirm: vi.fn(),
    handleOverwriteCancel: vi.fn(),
    handleYamlValidationChange: vi.fn(),
    resetConfigState: vi.fn(),
  }),
}))

vi.mock("@/lib/scheduled-scan-helpers", () => ({
  getNextCronExecutions: () => [],
  getConfigConflictMessage: () => null,
  validateScheduledScanStep: () => null,
}))

vi.mock("@/lib/workflow-config", () => ({
  adaptWorkflowProfile: (profile: { configuration: { steps: Record<string, unknown> } }, workflow: { name: string }) => ({
    scanWorkflow: workflow.name,
    steps: profile.configuration.steps,
  }),
  hasNoEnabledWorkflowSteps: () => false,
  serializeCanonicalWorkflowConfiguration: vi.fn(),
  serializeWorkflowProfileDraft: () => "steps:\n  discovery:\n    enabled: true",
}))

describe("useScheduledScanDialogState", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    stateMocks.currentStep = 3
    stateMocks.workflows = []
  })

  it("第 3 步资源校验失败时阻止进入第 4 步，修复后才允许继续", () => {
    const { result } = renderHook(() =>
      useScheduledScanDialogState({
        open: true,
        onOpenChange: vi.fn(),
        hasPreset: false,
        totalSteps: 4,
        locale: "zh",
        t: (key) => key,
      })
    )
    const validateAndReveal = vi.fn(() => false)
    result.current.configValidationRef.current = { validateAndReveal }

    act(() => result.current.handleNext())

    expect(validateAndReveal).toHaveBeenCalledTimes(1)
    expect(stateMocks.goToNextStep).not.toHaveBeenCalled()
    expect(result.current.currentStep).toBe(3)

    validateAndReveal.mockReturnValue(true)
    act(() => result.current.handleNext())

    expect(validateAndReveal).toHaveBeenCalledTimes(2)
    expect(stateMocks.goToNextStep).toHaveBeenCalledTimes(1)
  })

  it("新建定时扫描默认使用扫描快照，且允许显式切换来源", () => {
    const { result } = renderHook(() =>
      useScheduledScanDialogState({
        open: true,
        onOpenChange: vi.fn(),
        hasPreset: false,
        totalSteps: 4,
        locale: "zh",
        t: (key) => key,
      })
    )

    expect(result.current.inputSource).toBe("scanSnapshot")

    act(() => result.current.setInputSource("targetInventory"))
    expect(result.current.inputSource).toBe("targetInventory")
  })

  it("复用工作流选择器时从选中工作流的 Profile 初始化扫描配置", async () => {
    stateMocks.currentStep = 2
    stateMocks.workflows = [{
      name: "scanWorkflows/default",
      isExecutable: true,
      steps: [{ stepId: "discovery", engineId: "builtin.discovery" }],
    }]
    stateMocks.loadWorkflowProfile.mockResolvedValue({
      scanWorkflow: "scanWorkflows/default",
      configuration: { steps: { discovery: { enabled: true, engineConfig: { recon: { enabled: true } } } } },
    })

    const { result } = renderHook(() =>
      useScheduledScanDialogState({
        open: true,
        onOpenChange: vi.fn(),
        hasPreset: false,
        totalSteps: 4,
        locale: "zh",
        t: (key) => key,
      })
    )

    await act(async () => {
      await result.current.handleWorkflowNamesChange(["scanWorkflows/default"])
    })

    expect(stateMocks.loadWorkflowProfile).toHaveBeenCalledWith("scanWorkflows/default")
    expect(stateMocks.applyProfileConfiguration).toHaveBeenCalledWith("steps:\n  discovery:\n    enabled: true")
    expect(result.current.selectedScanWorkflowName).toBe("scanWorkflows/default")
  })
})
