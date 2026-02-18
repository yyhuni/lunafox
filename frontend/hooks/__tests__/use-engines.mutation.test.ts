import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  useCreateEngine,
  useDeleteEngine,
  useUpdateEngine,
} from "@/hooks/use-engines"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const engineServiceMocks = vi.hoisted(() => ({
  getPresetEngines: vi.fn(),
  getPresetEngine: vi.fn(),
  getEngines: vi.fn(),
  getEngine: vi.fn(),
  createEngine: vi.fn(),
  updateEngine: vi.fn(),
  deleteEngine: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/engine.service", () => ({
  getPresetEngines: engineServiceMocks.getPresetEngines,
  getPresetEngine: engineServiceMocks.getPresetEngine,
  getEngines: engineServiceMocks.getEngines,
  getEngine: engineServiceMocks.getEngine,
  createEngine: engineServiceMocks.createEngine,
  updateEngine: engineServiceMocks.updateEngine,
  deleteEngine: engineServiceMocks.deleteEngine,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

describe("use-engines mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("创建引擎成功时保留成功提示与列表失效", async () => {
    engineServiceMocks.createEngine.mockResolvedValue({
      id: 1,
      name: "default-engine",
      configuration: "name: default",
      createdAt: "2026-02-11T00:00:00Z",
      updatedAt: "2026-02-11T00:00:00Z",
    })
    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useCreateEngine(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        name: "default-engine",
        configuration: "name: default",
      })
    })

    expect(toastMocks.success).toHaveBeenCalledWith("toast.engine.create.success")
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["engines"],
    })
  })

  it("创建引擎失败时保留错误码映射与 fallback key", async () => {
    engineServiceMocks.createEngine.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "CONFLICT",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useCreateEngine())

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
      "toast.engine.create.error"
    )
  })

  it("更新引擎成功时失效列表与 detail key", async () => {
    engineServiceMocks.updateEngine.mockResolvedValue({
      id: 8,
      name: "patched",
      configuration: "name: patched",
      createdAt: "2026-02-11T00:00:00Z",
      updatedAt: "2026-02-11T00:00:00Z",
    })
    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUpdateEngine(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        id: 8,
        data: { name: "patched" },
      })
    })

    expect(engineServiceMocks.updateEngine).toHaveBeenCalledWith(8, { name: "patched" })
    expect(toastMocks.success).toHaveBeenCalledWith("toast.engine.update.success")
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["engines"],
    })
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["engines", 8],
    })
  })

  it("更新引擎失败时保留错误码映射与 fallback key", async () => {
    engineServiceMocks.updateEngine.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useUpdateEngine())

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
      "toast.engine.update.error"
    )
  })

  it("删除引擎成功时保留成功提示与列表失效", async () => {
    engineServiceMocks.deleteEngine.mockResolvedValue(undefined)

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useDeleteEngine(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(9)
    })

    expect(engineServiceMocks.deleteEngine).toHaveBeenCalledWith(9, expect.anything())
    expect(toastMocks.success).toHaveBeenCalledWith("toast.engine.delete.success")
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["engines"],
    })
  })

  it("删除引擎失败时保留错误码映射与 fallback key", async () => {
    engineServiceMocks.deleteEngine.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useDeleteEngine())

    await act(async () => {
      await expect(result.current.mutateAsync(9)).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.engine.delete.error"
    )
  })
})
