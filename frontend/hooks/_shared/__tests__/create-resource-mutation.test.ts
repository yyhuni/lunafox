import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { useResourceMutation } from "@/hooks/_shared/create-resource-mutation"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

describe("useResourceMutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("在查询失效完成后用同一个 toast id 将 loading 更新为成功", async () => {
    const mutationFn = vi.fn().mockResolvedValue({ ok: true })
    let resolveInvalidation: (() => void) | undefined
    const invalidation = new Promise<void>((resolve) => {
      resolveInvalidation = resolve
    })
    const onSuccess = vi.fn(({ toast }) => {
      toast.success("resource.update.success")
    })
    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries").mockReturnValue(invalidation)

    const { result } = renderHookWithProviders(
      () =>
        useResourceMutation({
          mutationFn,
          loadingToast: {
            key: "common.status.updating",
            params: {},
            id: (variables: { id: number }) => `update-${variables.id}`,
          },
          invalidate: [
            { queryKey: ["resource"] },
            ({ variables }) => ({ queryKey: ["resource", variables.id] }),
          ],
          onSuccess,
        }),
      { queryClient }
    )

    let mutationPromise: Promise<unknown> | undefined
    await act(async () => {
      mutationPromise = result.current.mutateAsync({ id: 7 })
      await Promise.resolve()
    })

    const [variables] = mutationFn.mock.calls[0]
    expect(variables).toEqual({ id: 7 })
    expect(toastMocks.loading).toHaveBeenCalledWith("common.status.updating", {}, "update-7")
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["resource"] })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["resource", 7] })
    expect(toastMocks.success).not.toHaveBeenCalled()
    expect(toastMocks.dismiss).not.toHaveBeenCalled()

    await act(async () => {
      resolveInvalidation?.()
      await mutationPromise
    })

    expect(toastMocks.success).toHaveBeenCalledWith("resource.update.success", undefined, "update-7")
    expect(toastMocks.dismiss).not.toHaveBeenCalled()
    expect(onSuccess).toHaveBeenCalled()
  })

  it("失败时用同一个 toast id 更新为默认错误码提示", async () => {
    const mutationFn = vi.fn().mockRejectedValue({
      response: {
        data: {
          error: {
            code: "NOT_FOUND",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() =>
      useResourceMutation({
        mutationFn,
        loadingToast: {
          key: "common.status.updating",
          params: {},
          id: "update-resource",
        },
        errorFallbackKey: "toast.generic.error",
      })
    )

    await act(async () => {
      await expect(result.current.mutateAsync({ id: 1 })).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "NOT_FOUND",
      "toast.generic.error",
      "update-resource"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalled()
  })

  it("提供 onError 时不会触发默认 errorFromCode", async () => {
    const mutationFn = vi.fn().mockRejectedValue(new Error("boom"))
    const onError = vi.fn(({ toast }: { toast: { error: (key: string) => void } }) => {
      toast.error("toast.custom.error")
    })

    const { result } = renderHookWithProviders(() =>
      useResourceMutation({
        mutationFn,
        loadingToast: {
          key: "common.status.saving",
          params: {},
          id: "save-resource",
        },
        onError,
      })
    )

    await act(async () => {
      await expect(result.current.mutateAsync({ id: 9 })).rejects.toBeDefined()
    })

    expect(onError).toHaveBeenCalled()
    expect(toastMocks.error).toHaveBeenCalledWith("toast.custom.error", undefined, "save-resource")
    expect(toastMocks.errorFromCode).not.toHaveBeenCalled()
    expect(toastMocks.dismiss).not.toHaveBeenCalled()
  })

  it("没有终态提示时会清理 loading toast", async () => {
    const mutationFn = vi.fn().mockResolvedValue({ ok: true })
    const { result } = renderHookWithProviders(() =>
      useResourceMutation({
        mutationFn,
        loadingToast: {
          key: "common.status.saving",
          params: {},
          id: "silent-save",
        },
      })
    )

    await act(async () => {
      await result.current.mutateAsync({ id: 2 })
    })

    expect(toastMocks.dismiss).toHaveBeenCalledWith("silent-save")
  })

  it("onMutate 抛错时仍用同一个 toast id 显示错误", async () => {
    const mutationFn = vi.fn()
    const { result } = renderHookWithProviders(() =>
      useResourceMutation({
        mutationFn,
        loadingToast: {
          key: "common.status.saving",
          params: {},
          id: "mutate-setup",
        },
        onMutate: () => {
          throw new Error("setup failed")
        },
      })
    )

    await act(async () => {
      await expect(result.current.mutateAsync({ id: 3 })).rejects.toThrow("setup failed")
    })

    expect(mutationFn).not.toHaveBeenCalled()
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      null,
      undefined,
      "mutate-setup"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalledWith("mutate-setup")
  })

  it("onSuccess 在终态提示前抛错时仍用同一个 toast id 显示错误", async () => {
    const mutationFn = vi.fn().mockResolvedValue({ ok: true })
    const { result } = renderHookWithProviders(() =>
      useResourceMutation({
        mutationFn,
        loadingToast: {
          key: "common.status.saving",
          params: {},
          id: "success-callback",
        },
        onSuccess: () => {
          throw new Error("success callback failed")
        },
      })
    )

    await act(async () => {
      await expect(result.current.mutateAsync({ id: 6 })).rejects.toThrow(
        "success callback failed"
      )
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      null,
      undefined,
      "success-callback"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalledWith("success-callback")
  })

  it("并发操作按各自 id 原位更新，即使后发操作先完成", async () => {
    const resolvers = new Map<number, (value: { id: number }) => void>()
    const mutationFn = vi.fn(
      ({ id }: { id: number }) =>
        new Promise<{ id: number }>((resolve) => {
          resolvers.set(id, resolve)
        })
    )
    const onSuccess = vi.fn(({ data, toast }) => {
      toast.success("resource.update.success", { id: data.id })
    })
    const { result } = renderHookWithProviders(() =>
      useResourceMutation({
        mutationFn,
        loadingToast: {
          key: "common.status.updating",
          params: {},
          id: ({ id }: { id: number }) => `update-${id}`,
        },
        onSuccess,
      })
    )

    let first: Promise<{ id: number }> | undefined
    let second: Promise<{ id: number }> | undefined
    await act(async () => {
      first = result.current.mutateAsync({ id: 1 })
      second = result.current.mutateAsync({ id: 2 })
      await Promise.resolve()
    })

    await act(async () => {
      resolvers.get(2)?.({ id: 2 })
      await second
      resolvers.get(1)?.({ id: 1 })
      await first
    })

    expect(toastMocks.success).toHaveBeenCalledWith(
      "resource.update.success",
      { id: 2 },
      "update-2"
    )
    expect(toastMocks.success).toHaveBeenCalledWith(
      "resource.update.success",
      { id: 1 },
      "update-1"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalled()
  })

  it("warning 终态继承 loading id", async () => {
    const mutationFn = vi.fn().mockResolvedValue({ partial: true })
    const { result } = renderHookWithProviders(() =>
      useResourceMutation({
        mutationFn,
        loadingToast: {
          key: "common.status.updating",
          params: {},
          id: "partial-update",
        },
        onSuccess: ({ toast }) => {
          toast.warning("resource.update.partial")
        },
      })
    )

    await act(async () => {
      await result.current.mutateAsync({ id: 4 })
    })

    expect(toastMocks.warning).toHaveBeenCalledWith(
      "resource.update.partial",
      undefined,
      "partial-update"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalledWith("partial-update")
  })

  it("显式不同 id 的独立提示不会冒充当前操作终态", async () => {
    const mutationFn = vi.fn().mockResolvedValue({ ok: true })
    const { result } = renderHookWithProviders(() =>
      useResourceMutation({
        mutationFn,
        loadingToast: {
          key: "common.status.saving",
          params: {},
          id: "save-resource",
        },
        onSuccess: ({ toast }) => {
          toast.success("background.sync.started", undefined, "background-sync")
        },
      })
    )

    await act(async () => {
      await result.current.mutateAsync({ id: 5 })
    })

    expect(toastMocks.success).toHaveBeenCalledWith(
      "background.sync.started",
      undefined,
      "background-sync"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("save-resource")
  })
})
