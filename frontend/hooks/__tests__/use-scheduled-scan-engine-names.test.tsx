import { renderHook } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { useScheduledScanEngineNames } from "@/hooks/use-scheduled-scan-engine-names"
import type { EngineCatalogSummary } from "@/types/engine-catalog.types"
import type { ScanWorkflow } from "@/types/scan-workflow.types"
import type { ScheduledScan } from "@/types/scheduled-scan.types"

const hookMocks = vi.hoisted(() => ({
  useEngineCatalog: vi.fn(),
  useScanWorkflows: vi.fn(),
}))

vi.mock("@/hooks/use-engine-catalog", () => ({
  useEngineCatalog: hookMocks.useEngineCatalog,
}))

vi.mock("@/hooks/use-scan-workflows", () => ({
  useScanWorkflows: hookMocks.useScanWorkflows,
}))

const engineId = "engine.lunafox.subdomain_discovery"

const workflow: ScanWorkflow = {
  name: "default",
  displayName: "Default",
  description: "",
  stages: [{ stageId: "discovery", steps: [{ stageId: "discovery", stepId: "subdomains", engineId, profileDefaultEnabled: true }] }],
  steps: [{ stageId: "discovery", stepId: "subdomains", engineId, profileDefaultEnabled: true }],
  isBuiltin: true,
  isExecutable: true,
  etag: "etag",
  createTime: "",
  updateTime: "",
}

const scheduledScan: ScheduledScan = {
  id: 1,
  name: "Default schedule",
  displayName: "Default schedule",
  scanWorkflow: workflow.name,
  organizationId: 1,
  organizationName: "Default organization",
  targetId: null,
  targetName: null,
  scanMode: "organization",
  inputSource: "scanSnapshot",
  cronExpression: "0 * * * *",
  isEnabled: true,
  nextRunTime: null,
  lastRunTime: null,
  runCount: 0,
  successfulHandoffCount: 0,
  failedHandoffCount: 0,
  createdAt: "2026-07-25T00:00:00.000Z",
  updatedAt: "2026-07-25T00:00:00.000Z",
}

const engine: EngineCatalogSummary = {
  name: "Subdomain Discovery",
  engineId,
  manifestVersion: "engine.v5",
  publisher: "lunafox",
  packageVersion: "1.0.0",
  artifactRef: "ghcr.io/lunafox/subdomain-discovery:1.0.0",
  packageDigest: "sha256:test",
  execution: {
    engineApiMajor: 5,
    supportedTargetTypes: ["domain"],
  },
  localeResources: {
    en: {
      engine: {
        displayName: "Subdomain Discovery",
      },
    },
  },
}

describe("useScheduledScanEngineNames", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    hookMocks.useScanWorkflows.mockReturnValue({ data: [], isSuccess: true })
    hookMocks.useEngineCatalog.mockReturnValue({ data: [], isSuccess: true })
  })

  it("does not reject a workflow while the engine catalog is pending, then resolves its name", () => {
    const state = {
      engines: [] as EngineCatalogSummary[],
      isEngineCatalogPending: true,
    }
    hookMocks.useScanWorkflows.mockReturnValue({ data: [workflow], isSuccess: true })
    hookMocks.useEngineCatalog.mockImplementation(() => ({
      data: state.engines,
      isPending: state.isEngineCatalogPending,
      isSuccess: !state.isEngineCatalogPending,
    }))

    const { result, rerender } = renderHook(() => useScheduledScanEngineNames([scheduledScan]))

    expect(result.current[0]?.engineNames).toEqual([])

    state.engines = [engine]
    state.isEngineCatalogPending = false
    rerender()

    expect(result.current[0]?.engineNames).toEqual(["Subdomain Discovery"])
  })

  it("rejects a workflow whose engine is still absent after both catalogs resolve", () => {
    hookMocks.useScanWorkflows.mockReturnValue({ data: [workflow], isSuccess: true })
    hookMocks.useEngineCatalog.mockReturnValue({ data: [], isSuccess: true })

    expect(() => renderHook(() => useScheduledScanEngineNames([scheduledScan]))).toThrow(
      `Workflow ${workflow.name} references unavailable engine ${engineId}`,
    )
  })
})
