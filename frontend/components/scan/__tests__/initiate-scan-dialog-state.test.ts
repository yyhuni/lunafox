import { act, renderHook, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { useInitiateScanDialogState } from "@/components/scan/initiate-scan-dialog-state"
import { getMockEngineCatalogDetail } from "@/mock/data/engine-catalog"
import type { EngineCatalogDetail } from "@/types/engine-catalog.types"
import type { ScanWorkflow, ScanWorkflowProfile } from "@/types/scan-workflow.types"

type Deferred<T> = {
  promise: Promise<T>
  resolve: (value: T) => void
  reject: (error: unknown) => void
}

function createDeferred<T>(): Deferred<T> {
  let resolve!: (value: T) => void
  let reject!: (error: unknown) => void
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve
    reject = promiseReject
  })
  return { promise, resolve, reject }
}

const hookMocks = vi.hoisted(() => ({
  loadWorkflowProfile: vi.fn(),
  workflows: [] as ScanWorkflow[],
  engineCatalog: [] as EngineCatalogDetail[],
  initiateScan: { isPending: false, mutateAsync: vi.fn() },
  bulkInitiateScan: { isPending: false, mutateAsync: vi.fn() },
}))

vi.mock("@/hooks/use-engine-catalog", () => ({
  useEngineCatalogDetails: () => ({
    data: hookMocks.engineCatalog,
    isLoading: false,
    isError: false,
    errors: [],
  }),
}))

vi.mock("@/hooks/use-scan-workflows", () => ({
  useScanWorkflows: () => ({
    data: hookMocks.workflows,
    isLoading: false,
    isError: false,
  }),
  useLoadScanWorkflowProfile: () => hookMocks.loadWorkflowProfile,
}))

vi.mock("@/hooks/use-scans", () => ({
  useInitiateScan: () => hookMocks.initiateScan,
  useBulkInitiateScan: () => hookMocks.bulkInitiateScan,
}))

describe("useInitiateScanDialogState", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    hookMocks.workflows = [
      createWorkflow("full", "完整扫描"),
      createWorkflow("quick", "快速侦查"),
    ]
    hookMocks.engineCatalog = [
      getMockEngineCatalogDetail("engine.lunafox.subdomain_discovery")!,
    ]
    hookMocks.initiateScan.mutateAsync.mockResolvedValue(undefined)
    hookMocks.bulkInitiateScan.mutateAsync.mockResolvedValue(undefined)
  })

  it("工作流详情加载期间先同步选中态，详情返回后再允许进入配置步骤", async () => {
    const fullProfile = createDeferred<ScanWorkflowProfile>()
    hookMocks.loadWorkflowProfile.mockReturnValue(fullProfile.promise)

    const { result } = renderHook(() =>
      useInitiateScanDialogState({
        targetId: 1,
        onOpenChange: vi.fn(),
        tToast: (key) => key,
      })
    )

    act(() => {
      void result.current.handleScanWorkflowNameChange(["full"])
    })

    expect(result.current.selectedWorkflowNames).toEqual(["full"])
    expect(result.current.canProceedToReview).toBe(false)

    await act(async () => {
      fullProfile.resolve({
        name: "scanWorkflows/full/profile",
        scanWorkflow: "scanWorkflows/full",
        configuration: {
          steps: {
            discovery: {
              enabled: false,
              engineConfig: {
                scope: { enabled: true },
              },
            },
          },
        },
      })
      await fullProfile.promise
    })

    await waitFor(() => {
      expect(result.current.canProceedToReview).toBe(true)
    })
    expect(result.current.configuration).toContain("discovery")
  })

  it("快速切换工作流时忽略较早返回的详情配置", async () => {
    const fullProfile = createDeferred<ScanWorkflowProfile>()
    const quickProfile = createDeferred<ScanWorkflowProfile>()
    hookMocks.loadWorkflowProfile.mockImplementation((name: string) => {
      return name === "full" ? fullProfile.promise : quickProfile.promise
    })

    const { result } = renderHook(() =>
      useInitiateScanDialogState({
        targetId: 1,
        onOpenChange: vi.fn(),
        tToast: (key) => key,
      })
    )

    act(() => {
      void result.current.handleScanWorkflowNameChange(["full"])
    })
    act(() => {
      void result.current.handleScanWorkflowNameChange(["quick"])
    })

    expect(result.current.selectedWorkflowNames).toEqual(["quick"])

    await act(async () => {
      quickProfile.resolve({
        name: "scanWorkflows/quick/profile",
        scanWorkflow: "scanWorkflows/quick",
        configuration: {
          steps: {
            discovery: {
              enabled: false,
              engineConfig: {
                quickOnly: { enabled: true },
              },
            },
          },
        },
      })
      await quickProfile.promise
    })

    await waitFor(() => {
      expect(result.current.configuration).toContain("quickOnly")
    })

    await act(async () => {
      fullProfile.resolve({
        name: "scanWorkflows/full/profile",
        scanWorkflow: "scanWorkflows/full",
        configuration: {
          steps: {
            discovery: {
              enabled: false,
              engineConfig: {
                fullOnly: { enabled: true },
              },
            },
          },
        },
      })
      await fullProfile.promise
    })

    expect(result.current.selectedWorkflowNames).toEqual(["quick"])
    expect(result.current.configuration).toContain("quickOnly")
    expect(result.current.configuration).not.toContain("fullOnly")
  })

  it("打开时默认选择第一个工作流并加载其配置", async () => {
    hookMocks.loadWorkflowProfile.mockResolvedValue({
      name: "scanWorkflows/full/profile",
      scanWorkflow: "scanWorkflows/full",
      configuration: { steps: { discovery: { enabled: false, engineConfig: { recon: { enabled: true, timeout: 60 } } } } },
    })

    const { result } = renderHook(() =>
      useInitiateScanDialogState({
        open: true,
        targetId: 1,
        onOpenChange: vi.fn(),
        tToast: (key) => key,
      })
    )

    await waitFor(() => {
      expect(result.current.selectedWorkflowNames).toEqual(["full"])
      expect(result.current.configuration).toContain("discovery")
    })
    expect(result.current.inputSource).toBe("scanSnapshot")
    expect(hookMocks.loadWorkflowProfile).toHaveBeenCalledWith("full")
  })

  it("当所有 Step 都关闭时禁止启动并暴露明确状态", async () => {
    hookMocks.loadWorkflowProfile.mockResolvedValue({
      name: "scanWorkflows/full/profile",
      scanWorkflow: "scanWorkflows/full",
      configuration: { steps: { discovery: { enabled: false, engineConfig: { recon: { enabled: true, timeout: 60 } } } } },
    })

    const { result } = renderHook(() =>
      useInitiateScanDialogState({
        open: true,
        targetId: 1,
        onOpenChange: vi.fn(),
        tToast: (key) => key,
      })
    )

    await waitFor(() => {
      expect(result.current.configuration).toContain("discovery")
    })

    act(() => {
      result.current.handleManualConfigChange("steps:\n  discovery:\n    enabled: false")
    })

    expect(result.current.hasNoEnabledSteps).toBe(true)
    expect(result.current.canStart).toBe(false)
  })

  it("资源字段校验失败时不提交，修复后才发起扫描请求", async () => {
    hookMocks.loadWorkflowProfile.mockResolvedValue({
      name: "scanWorkflows/full/profile",
      scanWorkflow: "scanWorkflows/full",
      configuration: {
        steps: {
          discovery: {
            enabled: false,
            engineConfig: {
              recon: { enabled: true, timeout: 60 },
            },
          },
        },
      },
    })

    const { result } = renderHook(() =>
      useInitiateScanDialogState({
        open: true,
        targetId: 1,
        onOpenChange: vi.fn(),
        tToast: (key) => key,
      })
    )

    await waitFor(() => expect(result.current.configuration).toContain("discovery"))
    act(() => {
      result.current.handleManualConfigChange(
        "steps:\n  discovery:\n    enabled: true\n    engineConfig:\n      recon:\n        enabled: true\n        timeout: 60\n        threads: 10\n      bruteforce:\n        enabled: false\n      resolve:\n        enabled: true\n        timeout: 3600\n        resolvers: resolvers.txt\n        threads: 100\n        rate-limit: 150\n        wildcard-filter: false"
      )
    })
    await waitFor(() => expect(result.current.canStart).toBe(true))
    const validateAndReveal = vi.fn(() => false)
    result.current.configValidationRef.current = { validateAndReveal }

    await act(async () => {
      await result.current.handleInitiate()
    })

    expect(validateAndReveal).toHaveBeenCalledTimes(1)
    expect(hookMocks.initiateScan.mutateAsync).not.toHaveBeenCalled()
    expect(hookMocks.bulkInitiateScan.mutateAsync).not.toHaveBeenCalled()

    validateAndReveal.mockReturnValue(true)
    act(() => result.current.setInputSource("targetInventory"))
    await act(async () => {
      await result.current.handleInitiate()
    })

    expect(validateAndReveal).toHaveBeenCalledTimes(2)
    expect(hookMocks.initiateScan.mutateAsync).toHaveBeenCalledTimes(1)
    expect(hookMocks.initiateScan.mutateAsync).toHaveBeenCalledWith(
      expect.objectContaining({ inputSource: "targetInventory" })
    )
  })
})

function createWorkflow(name: string, displayName: string): ScanWorkflow {
  return {
    name,
    displayName,
    description: "",
    stages: [{ stageId: "discovery", steps: [{ stageId: "discovery", stepId: "discovery", engineId: "engine.lunafox.subdomain_discovery", profileDefaultEnabled: true }] }],
    steps: [{ stageId: "discovery", stepId: "discovery", engineId: "engine.lunafox.subdomain_discovery", profileDefaultEnabled: true }],
    isBuiltin: false,
    isExecutable: true,
    etag: "etag",
    createTime: "2026-01-01T00:00:00Z",
    updateTime: "2026-01-01T00:00:00Z",
  }
}
