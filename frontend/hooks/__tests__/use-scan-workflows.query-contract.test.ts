import { waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { useScanWorkflowList, useScanWorkflowProfile } from "@/hooks/use-scan-workflows"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"

const service = vi.hoisted(() => ({ listScanWorkflows: vi.fn(), getScanWorkflowProfile: vi.fn() }))
vi.mock("@/services/scan-workflow.service", () => ({
  listScanWorkflows: service.listScanWorkflows,
  getScanWorkflowProfile: service.getScanWorkflowProfile,
  getScanWorkflow: vi.fn(), createScanWorkflow: vi.fn(), updateScanWorkflow: vi.fn(),
}))

describe("use-scan-workflows query contract", () => {
  beforeEach(() => vi.clearAllMocks())

  it("列表 key 绑定分页查询形状", async () => {
    service.listScanWorkflows.mockResolvedValue({ scanWorkflows: [], nextPageToken: "", totalSize: 0 })
    const { result } = renderHookWithProviders(() => useScanWorkflowList({ pageSize: 20, filter: "default" }))
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(service.listScanWorkflows).toHaveBeenCalledWith({ pageSize: 20, filter: "default" })
  })

  it("Profile 按工作流父资源读取", async () => {
    service.getScanWorkflowProfile.mockResolvedValue({ id: "scanWorkflows/default", name: "scanWorkflows/default", scanWorkflow: "scanWorkflows/default", configuration: { steps: { discover: { enabled: false, engineConfig: { recon: { enabled: true, timeout: 60 } } } } } })
    const { result } = renderHookWithProviders(() => useScanWorkflowProfile("default"))
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(service.getScanWorkflowProfile).toHaveBeenCalledWith("default")
  })
})
