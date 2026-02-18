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

  it("成功时会处理 loading、失效查询与成功回调", async () => {
    const mutationFn = vi.fn().mockResolvedValue({ ok: true })
    const onSuccess = vi.fn()
    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")

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

    await act(async () => {
      await result.current.mutateAsync({ id: 7 })
    })

    const [variables] = mutationFn.mock.calls[0]
    expect(variables).toEqual({ id: 7 })
    expect(toastMocks.loading).toHaveBeenCalledWith("common.status.updating", {}, "update-7")
    expect(toastMocks.dismiss).toHaveBeenCalledWith("update-7")
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["resource"] })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["resource", 7] })
    expect(onSuccess).toHaveBeenCalled()
  })

  it("失败时默认走 errorFromCode 映射", async () => {
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
        errorFallbackKey: "toast.generic.error",
      })
    )

    await act(async () => {
      await expect(result.current.mutateAsync({ id: 1 })).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith("NOT_FOUND", "toast.generic.error")
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

    expect(toastMocks.dismiss).toHaveBeenCalledWith("save-resource")
    expect(onError).toHaveBeenCalled()
    expect(toastMocks.error).toHaveBeenCalledWith("toast.custom.error")
    expect(toastMocks.errorFromCode).not.toHaveBeenCalled()
  })
})
