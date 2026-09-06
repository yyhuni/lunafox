import { renderHook } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { toastFeedback, useToastMessages } from "@/lib/toast-helpers"

const sonnerMocks = vi.hoisted(() => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    loading: vi.fn(),
    warning: vi.fn(),
    dismiss: vi.fn(),
  },
}))

vi.mock("sonner", () => ({
  toast: sonnerMocks.toast,
}))

describe("toast-helpers", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("toast.success 支持 toastId", () => {
    toastFeedback.success("done", { id: "toast-1" })
    expect(sonnerMocks.toast.success).toHaveBeenCalledWith("done", { id: "toast-1" })
  })

  it("useToastMessages.errorFromCode 走错误码映射", () => {
    const { result } = renderHook(() => useToastMessages())
    result.current.errorFromCode("NOT_FOUND")
    expect(sonnerMocks.toast.error).toHaveBeenCalledWith("errors.notFound")
  })

  it("useToastMessages.loading 会透传参数和 id", () => {
    const { result } = renderHook(() => useToastMessages())
    result.current.loading("common.status.deleting", { count: 3 }, "loading-1")
    expect(sonnerMocks.toast.loading).toHaveBeenCalledWith(
      'common.status.deleting:{"count":3}',
      { id: "loading-1" }
    )
  })

  it.each([
    ["success", "done"],
    ["error", "failed"],
    ["warning", "partial"],
  ] as const)("useToastMessages.%s 会把稳定 id 传给终态 toast", (method, key) => {
    const { result } = renderHook(() => useToastMessages())
    result.current[method](key, { count: 2 }, "operation-1")
    expect(sonnerMocks.toast[method]).toHaveBeenCalledWith(`${key}:{"count":2}`, {
      id: "operation-1",
    })
  })

  it("useToastMessages.errorFromCode 会把稳定 id 传给映射后的错误 toast", () => {
    const { result } = renderHook(() => useToastMessages())
    result.current.errorFromCode("NOT_FOUND", undefined, "operation-2")
    expect(sonnerMocks.toast.error).toHaveBeenCalledWith("errors.notFound", {
      id: "operation-2",
    })
  })
})
