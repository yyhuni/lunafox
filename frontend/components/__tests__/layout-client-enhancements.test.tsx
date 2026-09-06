import { render } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

const mockUseLocale = vi.fn<() => "zh" | "en">()

vi.mock("next/dynamic", () => ({
  default: () => () => null,
}))

vi.mock("next-intl", () => ({
  useLocale: () => mockUseLocale(),
}))

describe("LayoutClientEnhancements", () => {
  beforeEach(() => {
    mockUseLocale.mockReset()
    document.documentElement.lang = "zh-CN"
  })

  it("会根据当前 locale 同步 html lang，并在 locale 变化时更新", async () => {
    mockUseLocale.mockReturnValue("zh")

    const { LayoutClientEnhancements } = await import("@/components/layout-client-enhancements")
    const { rerender } = render(<LayoutClientEnhancements />)

    expect(document.documentElement.lang).toBe("zh-CN")

    mockUseLocale.mockReturnValue("en")
    rerender(<LayoutClientEnhancements />)

    expect(document.documentElement.lang).toBe("en")
  })
})
