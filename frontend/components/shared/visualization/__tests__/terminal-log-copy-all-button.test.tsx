import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { TerminalLogCopyAllButton } from "@/components/shared/visualization/terminal-log-copy-all-button"

const sonnerMocks = vi.hoisted(() => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}))

vi.mock("sonner", () => ({
  toast: sonnerMocks.toast,
}))

describe("TerminalLogCopyAllButton", () => {
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

  it("preserves copy behavior when composed as a tooltip trigger", async () => {
    render(
      <TerminalLogCopyAllButton
        value={"first log\nsecond log"}
        copyLabel="复制全部日志"
        copiedLabel="已复制"
        toastId="terminal-log-copy"
      />
    )

    fireEvent.click(screen.getByRole("button", { name: "复制全部日志" }))

    await waitFor(() => {
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith("first log\nsecond log")
      expect(sonnerMocks.toast.success).toHaveBeenCalledWith("已复制", { id: "terminal-log-copy" })
    })

    expect(screen.getByRole("button", { name: "已复制" })).toBeInTheDocument()
  })
})
