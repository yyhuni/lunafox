import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { RuntimeConfigurationPanel } from "@/components/scan/history/scan-runtime-detail-drawer"

const sonnerMocks = vi.hoisted(() => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}))

vi.mock("sonner", () => ({
  toast: sonnerMocks.toast,
}))

vi.mock("next-intl", () => ({
  useLocale: () => "zh",
  useTranslations: (namespace: string) => (key: string) => {
    if (namespace === "toast" && key === "copyFailed") return "复制失败"
    return key
  },
}))

const labels: Record<string, string> = {
  copied: "已复制",
  noConfig: "暂无配置信息",
  "runtimeDrawer.details.copyConfig": "复制扫描配置",
}

const translate = (key: string) => labels[key] ?? key

describe("RuntimeConfigurationPanel", () => {
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

  it("copies the complete serialized scan configuration", async () => {
    const yamlContent = "steps:\n  directory_scan:\n    enabled: true"

    render(
      <RuntimeConfigurationPanel
        yamlContent={yamlContent}
        t={translate as React.ComponentProps<typeof RuntimeConfigurationPanel>["t"]}
      />
    )

    fireEvent.click(screen.getByRole("button", { name: "复制扫描配置" }))

    await waitFor(() => {
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(yamlContent)
      expect(sonnerMocks.toast.success).toHaveBeenCalledWith("已复制", {
        id: "scan-runtime-configuration-copy",
      })
    })

    expect(screen.getByRole("button", { name: "已复制" })).toBeInTheDocument()
  })
})
