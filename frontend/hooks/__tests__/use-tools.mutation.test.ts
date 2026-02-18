import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  toolKeys,
  useCreateTool,
  useDeleteTool,
  useUpdateTool,
} from "@/hooks/use-tools"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const toolServiceMocks = vi.hoisted(() => ({
  getTools: vi.fn(),
  createTool: vi.fn(),
  updateTool: vi.fn(),
  deleteTool: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/tool.service", () => ({
  ToolService: toolServiceMocks,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

describe("use-tools mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("创建工具成功时保留 loading/success 与失效范围", async () => {
    toolServiceMocks.createTool.mockResolvedValue({
      tool: {
        id: 1,
        name: "nuclei",
      },
    })
    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useCreateTool(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        name: "nuclei",
        type: "opensource",
      })
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.creating",
      {},
      "create-tool"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("create-tool")
    expect(toastMocks.success).toHaveBeenCalledWith("toast.tool.create.success")
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: toolKeys.all,
      refetchType: "active",
    })
  })

  it("创建工具失败时保留错误码映射与回退 key", async () => {
    toolServiceMocks.createTool.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "CONFLICT",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useCreateTool())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          name: "nuclei",
          type: "opensource",
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.creating",
      {},
      "create-tool"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("create-tool")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "CONFLICT",
      "toast.tool.create.error"
    )
  })

  it("更新工具成功时保留 loading/success 与失效范围", async () => {
    toolServiceMocks.updateTool.mockResolvedValue({
      tool: {
        id: 7,
        name: "patched",
      },
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUpdateTool(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        id: 7,
        data: {
          name: "patched",
        },
      })
    })

    expect(toolServiceMocks.updateTool).toHaveBeenCalledWith(7, { name: "patched" })
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.updating",
      {},
      "update-tool"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("update-tool")
    expect(toastMocks.success).toHaveBeenCalledWith("toast.tool.update.success")
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: toolKeys.all,
      refetchType: "active",
    })
  })

  it("更新工具失败时保留错误码映射与回退 key", async () => {
    toolServiceMocks.updateTool.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useUpdateTool())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          id: 7,
          data: {
            name: "patched",
          },
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.updating",
      {},
      "update-tool"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("update-tool")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.tool.update.error"
    )
  })

  it("删除工具成功时保留 loading/success 与失效范围", async () => {
    toolServiceMocks.deleteTool.mockResolvedValue(undefined)

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useDeleteTool(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(99)
    })

    expect(toolServiceMocks.deleteTool).toHaveBeenCalledWith(99)
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.deleting",
      {},
      "delete-tool"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("delete-tool")
    expect(toastMocks.success).toHaveBeenCalledWith("toast.tool.delete.success")
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: toolKeys.all,
      refetchType: "active",
    })
  })

  it("删除工具失败时保留错误码映射和回退 key", async () => {
    toolServiceMocks.deleteTool.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useDeleteTool())

    await act(async () => {
      await expect(result.current.mutateAsync(99)).rejects.toBeDefined()
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.deleting",
      {},
      "delete-tool"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("delete-tool")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.tool.delete.error"
    )
  })
})
