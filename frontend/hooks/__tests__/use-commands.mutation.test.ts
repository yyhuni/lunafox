import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  commandKeys,
  useBatchDeleteCommands,
  useCreateCommand,
  useDeleteCommand,
  useUpdateCommand,
} from "@/hooks/use-commands"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const commandServiceMocks = vi.hoisted(() => ({
  getCommands: vi.fn(),
  getCommandById: vi.fn(),
  createCommand: vi.fn(),
  updateCommand: vi.fn(),
  deleteCommand: vi.fn(),
  batchDeleteCommands: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/command.service", () => ({
  CommandService: commandServiceMocks,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

describe("use-commands mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("创建命令成功时保留成功提示与列表失效", async () => {
    commandServiceMocks.createCommand.mockResolvedValue({
      command: {
        id: 11,
        toolId: 1,
        name: "subdomain_scan",
        displayName: "Subdomain Scan",
        description: "scan",
        commandTemplate: "subfinder -d {{domain}}",
        createdAt: "2026-02-11T12:00:00Z",
        updatedAt: "2026-02-11T12:00:00Z",
      },
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useCreateCommand(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        toolId: 1,
        name: "subdomain_scan",
        displayName: "Subdomain Scan",
        commandTemplate: "subfinder -d {{domain}}",
      })
    })

    expect(commandServiceMocks.createCommand).toHaveBeenCalledWith({
      toolId: 1,
      name: "subdomain_scan",
      displayName: "Subdomain Scan",
      commandTemplate: "subfinder -d {{domain}}",
    })
    expect(toastMocks.success).toHaveBeenCalledWith("toast.command.create.success")
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: commandKeys.all })
  })

  it("更新命令成功时失效列表与详情缓存", async () => {
    commandServiceMocks.updateCommand.mockResolvedValue({
      command: {
        id: 11,
        toolId: 1,
        name: "subdomain_scan",
        displayName: "Subdomain Scan V2",
        description: "scan",
        commandTemplate: "subfinder -d {{domain}} -silent",
        createdAt: "2026-02-11T12:00:00Z",
        updatedAt: "2026-02-11T12:10:00Z",
      },
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUpdateCommand(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        id: 11,
        data: {
          displayName: "Subdomain Scan V2",
          commandTemplate: "subfinder -d {{domain}} -silent",
        },
      })
    })

    expect(commandServiceMocks.updateCommand).toHaveBeenCalledWith(11, {
      displayName: "Subdomain Scan V2",
      commandTemplate: "subfinder -d {{domain}} -silent",
    })
    expect(toastMocks.success).toHaveBeenCalledWith("toast.command.update.success")
    expect(invalidateSpy).toHaveBeenNthCalledWith(1, { queryKey: commandKeys.all })
    expect(invalidateSpy).toHaveBeenNthCalledWith(2, { queryKey: commandKeys.details() })
  })

  it("删除命令失败时保留错误码映射与 fallback key", async () => {
    commandServiceMocks.deleteCommand.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useDeleteCommand())

    await act(async () => {
      await expect(result.current.mutateAsync(99)).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.command.delete.error"
    )
    expect(toastMocks.success).not.toHaveBeenCalled()
  })

  it("删除命令成功时保留 success 提示与列表失效", async () => {
    commandServiceMocks.deleteCommand.mockResolvedValue(undefined)

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useDeleteCommand(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(99)
    })

    expect(commandServiceMocks.deleteCommand).toHaveBeenCalledWith(99)
    expect(toastMocks.success).toHaveBeenCalledWith("toast.command.delete.success")
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: commandKeys.all })
  })

  it("创建命令失败时保留错误码映射与 fallback key", async () => {
    commandServiceMocks.createCommand.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useCreateCommand())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          toolId: 1,
          name: "subdomain_scan",
          displayName: "Subdomain Scan",
          commandTemplate: "subfinder -d {{domain}}",
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.command.create.error"
    )
  })

  it("更新命令失败时保留错误码映射与 fallback key", async () => {
    commandServiceMocks.updateCommand.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useUpdateCommand())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          id: 11,
          data: {
            displayName: "Subdomain Scan V2",
            commandTemplate: "subfinder -d {{domain}} -silent",
          },
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.command.update.error"
    )
  })

  it("批量删除命令成功时保留计数提示与列表失效", async () => {
    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useBatchDeleteCommands(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync([1, 2, 999])
    })

    expect(toastMocks.success).toHaveBeenCalledWith("toast.command.delete.bulkSuccess", {
      count: 2,
    })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: commandKeys.all })
  })
})
