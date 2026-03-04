import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  useCreateWorkflow,
  useDeleteWorkflow,
  useUpdateWorkflow,
} from "@/hooks/use-workflows"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const engineServiceMocks = vi.hoisted(() => ({
  getPresetWorkflows: vi.fn(),
  getPresetWorkflow: vi.fn(),
  getWorkflows: vi.fn(),
  getWorkflow: vi.fn(),
  createWorkflow: vi.fn(),
  updateWorkflow: vi.fn(),
  deleteWorkflow: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/workflow.service", () => ({
  getPresetWorkflows: engineServiceMocks.getPresetWorkflows,
  getPresetWorkflow: engineServiceMocks.getPresetWorkflow,
  getWorkflows: engineServiceMocks.getWorkflows,
  getWorkflow: engineServiceMocks.getWorkflow,
  createWorkflow: engineServiceMocks.createWorkflow,
  updateWorkflow: engineServiceMocks.updateWorkflow,
  deleteWorkflow: engineServiceMocks.deleteWorkflow,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

describe("use-workflows mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("创建工作流成功时保留成功提示与列表失效", async () => {
    engineServiceMocks.createWorkflow.mockResolvedValue({
      id: 1,
      name: "default-engine",
      configuration: "name: default",
      createdAt: "2026-02-11T00:00:00Z",
      updatedAt: "2026-02-11T00:00:00Z",
    })
    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useCreateWorkflow(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        name: "default-engine",
        configuration: "name: default",
      })
    })

    expect(toastMocks.success).toHaveBeenCalledWith("toast.workflow.create.success")
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["workflows"],
    })
  })

  it("创建工作流失败时保留错误码映射与 fallback key", async () => {
    engineServiceMocks.createWorkflow.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "CONFLICT",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useCreateWorkflow())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          name: "default-engine",
          configuration: "name: default",
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "CONFLICT",
      "toast.workflow.create.error"
    )
  })

  it("更新工作流成功时失效列表与 detail key", async () => {
    engineServiceMocks.updateWorkflow.mockResolvedValue({
      id: 8,
      name: "patched",
      configuration: "name: patched",
      createdAt: "2026-02-11T00:00:00Z",
      updatedAt: "2026-02-11T00:00:00Z",
    })
    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUpdateWorkflow(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        id: 8,
        data: { name: "patched" },
      })
    })

    expect(engineServiceMocks.updateWorkflow).toHaveBeenCalledWith(8, { name: "patched" })
    expect(toastMocks.success).toHaveBeenCalledWith("toast.workflow.update.success")
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["workflows"],
    })
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["workflows", 8],
    })
  })

  it("更新工作流失败时保留错误码映射与 fallback key", async () => {
    engineServiceMocks.updateWorkflow.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useUpdateWorkflow())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          id: 8,
          data: { name: "patched" },
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.workflow.update.error"
    )
  })

  it("删除工作流成功时保留成功提示与列表失效", async () => {
    engineServiceMocks.deleteWorkflow.mockResolvedValue(undefined)

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useDeleteWorkflow(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(9)
    })

    expect(engineServiceMocks.deleteWorkflow).toHaveBeenCalledWith(9, expect.anything())
    expect(toastMocks.success).toHaveBeenCalledWith("toast.workflow.delete.success")
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["workflows"],
    })
  })

  it("删除工作流失败时保留错误码映射与 fallback key", async () => {
    engineServiceMocks.deleteWorkflow.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useDeleteWorkflow())

    await act(async () => {
      await expect(result.current.mutateAsync(9)).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.workflow.delete.error"
    )
  })
})
