import { act, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import {
  formatPageRefreshTimestamp,
  PageRefreshStatusButton,
} from "@/components/common/page-refresh-status-button"

const defaultProps = {
  isRefreshing: false,
  lastRefreshedAt: null,
  onRefresh: vi.fn(),
  refreshLabel: "刷新页面",
  updatedAtLabel: "更新时间：",
  updatedAtUnavailable: "暂未更新",
}

describe("PageRefreshStatusButton", () => {
  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it("formats completion time as a fixed local timestamp", () => {
    const completedAt = new Date(2026, 6, 24, 20, 9, 32)

    expect(formatPageRefreshTimestamp(completedAt)).toBe("2026/07/24 20:09:32")

    render(<PageRefreshStatusButton {...defaultProps} lastRefreshedAt={completedAt} />)

    expect(screen.getByRole("button", { name: "刷新页面" })).toHaveTextContent("更新时间：2026/07/24 20:09:32")
  })

  it("shows the unavailable state before the first manual refresh", () => {
    render(<PageRefreshStatusButton {...defaultProps} />)

    expect(screen.getByRole("button", { name: "刷新页面" })).toHaveTextContent("更新时间：暂未更新")
  })

  it("keeps feedback visible and rejects duplicate activation", () => {
    vi.useFakeTimers()
    const onRefresh = vi.fn(() => Promise.resolve())

    render(<PageRefreshStatusButton {...defaultProps} onRefresh={onRefresh} />)

    const button = screen.getByRole("button", { name: "刷新页面" })
    fireEvent.click(button)
    fireEvent.click(button)

    expect(onRefresh).toHaveBeenCalledTimes(1)
    expect(button).toBeDisabled()
    expect(button).toHaveAttribute("aria-busy", "true")

    act(() => {
      vi.advanceTimersByTime(300)
    })

    expect(button).toBeEnabled()
    expect(button).toHaveAttribute("aria-busy", "false")
  })

  it("renders a compact icon-only entry through the same owner", () => {
    render(<PageRefreshStatusButton {...defaultProps} display="icon-only" />)

    const button = screen.getByRole("button", { name: "刷新页面" })
    expect(button).toHaveTextContent("刷新页面")
    expect(button).not.toHaveTextContent("更新时间")
  })
})
