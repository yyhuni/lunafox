import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  workerKeys,
  useCreateWorker,
  useDeployWorker,
  useUpdateWorker,
  useDeleteWorker,
  useRestartWorker,
  useStopWorker,
} from "@/hooks/use-workers"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const workerServiceMocks = vi.hoisted(() => ({
  getWorkers: vi.fn(),
  getWorker: vi.fn(),
  createWorker: vi.fn(),
  updateWorker: vi.fn(),
  deleteWorker: vi.fn(),
  deployWorker: vi.fn(),
  restartWorker: vi.fn(),
  stopWorker: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/worker.service", () => ({
  workerService: workerServiceMocks,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

describe("use-workers mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("创建 worker 成功时保留成功提示与列表失效范围", async () => {
    workerServiceMocks.createWorker.mockResolvedValue({
      id: 1,
      name: "agent-1",
      ipAddress: "10.0.0.1",
      sshPort: 22,
      username: "root",
      status: "online",
      isLocal: false,
      createdAt: "2026-02-11T00:00:00Z",
    })
    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useCreateWorker(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        name: "agent-1",
        ipAddress: "10.0.0.1",
        password: "secret",
      })
    })

    expect(workerServiceMocks.createWorker).toHaveBeenCalledWith({
      name: "agent-1",
      ipAddress: "10.0.0.1",
      password: "secret",
    })
    expect(toastMocks.success).toHaveBeenCalledWith("toast.worker.create.success")
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: workerKeys.lists() })
  })

  it("部署 worker 失败时保留错误码映射与回退 key", async () => {
    workerServiceMocks.deployWorker.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "RATE_LIMITED",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useDeployWorker())

    await act(async () => {
      await expect(result.current.mutateAsync(7)).rejects.toBeDefined()
    })

    expect(workerServiceMocks.deployWorker).toHaveBeenCalledWith(7)
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "RATE_LIMITED",
      "toast.worker.deploy.error"
    )
  })

  it("更新 worker 成功时失效列表与详情缓存", async () => {
    workerServiceMocks.updateWorker.mockResolvedValue({
      id: 7,
      name: "agent-7",
      status: "online",
      createdAt: "2026-02-11T00:00:00Z",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUpdateWorker(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        id: 7,
        data: {
          name: "agent-7",
        },
      })
    })

    expect(workerServiceMocks.updateWorker).toHaveBeenCalledWith(7, {
      name: "agent-7",
    })
    expect(toastMocks.success).toHaveBeenCalledWith("toast.worker.update.success")
    expect(invalidateSpy).toHaveBeenNthCalledWith(1, {
      queryKey: workerKeys.lists(),
    })
    expect(invalidateSpy).toHaveBeenNthCalledWith(2, {
      queryKey: workerKeys.detail(7),
    })
  })

  it("更新 worker 失败时保留错误码映射与回退 key", async () => {
    workerServiceMocks.updateWorker.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useUpdateWorker())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          id: 7,
          data: {
            name: "agent-7",
          },
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.worker.update.error"
    )
  })

  it("删除 worker 成功时保留 success 提示与列表失效范围", async () => {
    workerServiceMocks.deleteWorker.mockResolvedValue(undefined)

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useDeleteWorker(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(5)
    })

    expect(workerServiceMocks.deleteWorker).toHaveBeenCalledWith(5)
    expect(toastMocks.success).toHaveBeenCalledWith("toast.worker.delete.success")
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: workerKeys.lists(),
      refetchType: "active",
    })
  })

  it("删除 worker 失败时保留错误码映射与回退 key", async () => {
    workerServiceMocks.deleteWorker.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useDeleteWorker())

    await act(async () => {
      await expect(result.current.mutateAsync(5)).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.worker.delete.error"
    )
  })

  it("部署 worker 成功时保留 success 提示与失效范围", async () => {
    workerServiceMocks.deployWorker.mockResolvedValue({
      id: 9,
      status: "online",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useDeployWorker(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(9)
    })

    expect(workerServiceMocks.deployWorker).toHaveBeenCalledWith(9)
    expect(toastMocks.success).toHaveBeenCalledWith("toast.worker.deploy.success")
    expect(invalidateSpy).toHaveBeenNthCalledWith(1, {
      queryKey: workerKeys.detail(9),
    })
    expect(invalidateSpy).toHaveBeenNthCalledWith(2, {
      queryKey: workerKeys.lists(),
    })
  })

  it("重启 worker 成功时保留 success 提示与失效范围", async () => {
    workerServiceMocks.restartWorker.mockResolvedValue({
      id: 10,
      status: "restarting",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useRestartWorker(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(10)
    })

    expect(workerServiceMocks.restartWorker).toHaveBeenCalledWith(10)
    expect(toastMocks.success).toHaveBeenCalledWith("toast.worker.restart.success")
    expect(invalidateSpy).toHaveBeenNthCalledWith(1, {
      queryKey: workerKeys.detail(10),
    })
    expect(invalidateSpy).toHaveBeenNthCalledWith(2, {
      queryKey: workerKeys.lists(),
    })
  })

  it("重启 worker 失败时保留错误码映射与回退 key", async () => {
    workerServiceMocks.restartWorker.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useRestartWorker())

    await act(async () => {
      await expect(result.current.mutateAsync(10)).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.worker.restart.error"
    )
  })

  it("停止 worker 成功时保留 success 提示与失效范围", async () => {
    workerServiceMocks.stopWorker.mockResolvedValue({
      id: 12,
      status: "stopped",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useStopWorker(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(12)
    })

    expect(workerServiceMocks.stopWorker).toHaveBeenCalledWith(12)
    expect(toastMocks.success).toHaveBeenCalledWith("toast.worker.stop.success")
    expect(invalidateSpy).toHaveBeenNthCalledWith(1, {
      queryKey: workerKeys.detail(12),
    })
    expect(invalidateSpy).toHaveBeenNthCalledWith(2, {
      queryKey: workerKeys.lists(),
    })
  })

  it("停止 worker 失败时保留错误码映射与回退 key", async () => {
    workerServiceMocks.stopWorker.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useStopWorker())

    await act(async () => {
      await expect(result.current.mutateAsync(12)).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.worker.stop.error"
    )
  })
})
