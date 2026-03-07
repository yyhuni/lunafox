import { waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  useWorkflowProfile,
  useWorkflowProfiles,
  useWorkflows,
} from "@/hooks/use-workflows"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"

const workflowServiceMocks = vi.hoisted(() => ({
  getWorkflowProfiles: vi.fn(),
  getWorkflowProfile: vi.fn(),
  getWorkflows: vi.fn(),
}))

vi.mock("@/services/workflow.service", () => ({
  getWorkflowProfiles: workflowServiceMocks.getWorkflowProfiles,
  getWorkflowProfile: workflowServiceMocks.getWorkflowProfile,
  getWorkflows: workflowServiceMocks.getWorkflows,
}))

describe("use-workflows query contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("workflows 查询返回只读元数据契约", async () => {
    workflowServiceMocks.getWorkflows.mockResolvedValue([
      {
        name: "subdomain_discovery",
        title: "Subdomain Discovery",
        description: "Discover subdomains",
        version: "1.0.0",
        configuration: {
          subdomain_discovery: {
            recon: { enabled: true },
          },
        },
      },
    ])

    const { result } = renderHookWithProviders(() => useWorkflows())

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(workflowServiceMocks.getWorkflows).toHaveBeenCalledTimes(1)
    expect(result.current.data?.[0]).toMatchObject({
      name: "subdomain_discovery",
      title: "Subdomain Discovery",
      description: "Discover subdomains",
      version: "1.0.0",
      configuration: {
        subdomain_discovery: {
          recon: { enabled: true },
        },
      },
    })
  })

  it("preset 查询返回 profile 契约（包含 workflowIds 和对象配置）", async () => {
    workflowServiceMocks.getWorkflowProfiles.mockResolvedValue([
      {
        id: "subdomain_discovery.fast",
        name: "Fast Profile",
        workflowIds: ["subdomain_discovery"],
        configuration: {
          subdomain_discovery: {
            recon: { enabled: false },
          },
        },
      },
    ])

    const { result } = renderHookWithProviders(() => useWorkflowProfiles())

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.[0]).toMatchObject({
      id: "subdomain_discovery.fast",
      workflowIds: ["subdomain_discovery"],
      configuration: {
        subdomain_discovery: {
          recon: { enabled: false },
        },
      },
    })
  })

  it("preset 详情查询按 id 读取", async () => {
    workflowServiceMocks.getWorkflowProfile.mockResolvedValue({
      id: "subdomain_discovery.default",
      name: "Default Profile",
      workflowIds: ["subdomain_discovery"],
      configuration: {
        subdomain_discovery: {
          recon: { enabled: false },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useWorkflowProfile("subdomain_discovery.default"))

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(workflowServiceMocks.getWorkflowProfile).toHaveBeenCalledWith("subdomain_discovery.default")
    expect(result.current.data?.workflowIds).toEqual(["subdomain_discovery"])
  })
})
