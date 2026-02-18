import { renderHook } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { showToast, useToastMessages } from "@/lib/toast-helpers"

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

  it("showToast.success 支持 toastId", () => {
    showToast.success("done", "toast-1")
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
})
