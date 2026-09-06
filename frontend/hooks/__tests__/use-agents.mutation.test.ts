import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  agentKeys,
  useCreateRegistrationToken,
  useDeleteAgent,
  useUpdateAgentConfig,
} from "@/hooks/use-agents"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const agentServiceMocks = vi.hoisted(() => ({
  getAgents: vi.fn(),
  getAgent: vi.fn(),
  deleteAgent: vi.fn(),
  updateDistributedConfig: vi.fn(),
  createRegistrationToken: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/agent.service", () => ({
  agentService: agentServiceMocks,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

describe("use-agents mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("创建注册令牌失败时保留错误码映射和 toast id", async () => {
    agentServiceMocks.createRegistrationToken.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "RATE_LIMITED",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useCreateRegistrationToken())

    await act(async () => {
      await expect(result.current.mutateAsync()).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "RATE_LIMITED",
      "toast.agent.token.error",
      "agent-token"
    )
  })

  it("创建注册令牌成功时保留成功提示和 toast id", async () => {
    agentServiceMocks.createRegistrationToken.mockResolvedValue({
      token: "token",
      expiresIn: 3600,
    })

    const { result } = renderHookWithProviders(() => useCreateRegistrationToken())

    await act(async () => {
      await result.current.mutateAsync()
    })

    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.agent.token.success",
      {},
      "agent-token"
    )
  })

  it("更新分布式配置成功时失效列表和 detail", async () => {
    agentServiceMocks.updateDistributedConfig.mockResolvedValue({
      id: 5,
      name: "agent-node-5",
    })
    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUpdateAgentConfig(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        id: 5,
        data: {
          maxTasks: 4,
        },
      })
    })

    expect(agentServiceMocks.updateDistributedConfig).toHaveBeenCalledWith(5, {
      maxTasks: 4,
    })
    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.agent.config.success",
      undefined,
      "update-agent-config-5"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: agentKeys.lists(),
    })
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: agentKeys.detail("agents/5"),
    })
  })

  it("更新分布式配置失败时保留错误码映射与回退 key", async () => {
    agentServiceMocks.updateDistributedConfig.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useUpdateAgentConfig())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          id: 5,
          data: {
            maxTasks: 4,
          },
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.agent.config.error",
      "update-agent-config-5"
    )
  })

  it("删除分布式节点成功时提示成功并失效列表", async () => {
    agentServiceMocks.deleteAgent.mockResolvedValue({
      message: "ok",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useDeleteAgent(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(9)
    })

    expect(agentServiceMocks.deleteAgent).toHaveBeenCalledWith(9)
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.deleting",
      {},
      "delete-agent-node-9"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalledWith("delete-agent-node-9")
    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.agent.delete.success",
      undefined,
      "delete-agent-node-9"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: agentKeys.lists(),
      refetchType: "active",
    })
  })

  it("删除分布式节点失败时保留错误码映射与回退 key", async () => {
    agentServiceMocks.deleteAgent.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useDeleteAgent())

    await act(async () => {
      await expect(result.current.mutateAsync(9)).rejects.toBeDefined()
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.deleting",
      {},
      "delete-agent-node-9"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalledWith("delete-agent-node-9")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.agent.delete.error",
      "delete-agent-node-9"
    )
  })
})
