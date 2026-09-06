import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { CopyButton } from "@/components/shared/feedback/copy-button"

const sonnerMocks = vi.hoisted(() => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}))

vi.mock("sonner", () => ({
  toast: sonnerMocks.toast,
}))

describe("CopyButton", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.defineProperty(window, "isSecureContext", {
      configurable: true,
      value: true,
    })
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    })
  })

  it("copies with the shared quiet icon affordance and stable toast id", async () => {
    render(
      <div className="group">
        <CopyButton
          value="https://acme.com/search?q=test"
          copyLabel="点击复制"
          copiedLabel="已复制"
          copyFailedLabel="复制失败"
          toastId="copy-target"
          hideUntilHover
        />
      </div>
    )

    const button = screen.getByRole("button", { name: "点击复制" })

    expect(button.className).toContain("hover:bg-transparent")
    expect(button.className).toContain("md:group-hover:opacity-100")

    fireEvent.click(button)

    await waitFor(() => {
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith("https://acme.com/search?q=test")
      expect(sonnerMocks.toast.success).toHaveBeenCalledWith("已复制", { id: "copy-target" })
    })

    expect(screen.getByRole("button", { name: "已复制" })).toBeInTheDocument()
  })
})
