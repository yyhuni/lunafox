import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  handleMutationError,
  handleSuccess,
  handleWarning,
} from "@/lib/error-handler"

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/lib/toast-helpers", () => ({
  toastFeedback: toastMocks,
}))

describe("error-handler toast lifecycle", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("用同一个 id 将 loading 更新为 success", () => {
    handleSuccess({}, "saved", "save-resource")
    expect(toastMocks.success).toHaveBeenCalledWith("saved", { id: "save-resource" })
    expect(toastMocks.dismiss).not.toHaveBeenCalled()
  })

  it("用同一个 id 将 loading 更新为 error", () => {
    handleMutationError(new Error("failed"), "save failed", "save-resource")
    expect(toastMocks.error).toHaveBeenCalledWith("save failed", { id: "save-resource" })
    expect(toastMocks.dismiss).not.toHaveBeenCalled()
  })

  it("用同一个 id 将 loading 更新为 warning", () => {
    handleWarning({}, "partially saved", "save-resource")
    expect(toastMocks.warning).toHaveBeenCalledWith("partially saved", { id: "save-resource" })
    expect(toastMocks.dismiss).not.toHaveBeenCalled()
  })
})
