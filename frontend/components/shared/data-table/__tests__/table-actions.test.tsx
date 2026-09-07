import { act, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { TableActions } from "@/components/shared/data-table/table-actions"

describe("TableActions", () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it("刷新中只显示一个旋转的刷新图标", () => {
    render(
      <TableActions
        selectedCount={0}
        onRefresh={vi.fn()}
        refreshLabel="刷新"
        isRefreshing
        showAddButton={false}
        showBulkAdd={false}
      />
    )

    const refreshButton = screen.getByRole("button", { name: "刷新" })
    const icons = refreshButton.querySelectorAll("svg")

    expect(refreshButton).toHaveAttribute("aria-busy", "true")
    expect(refreshButton).toHaveClass("size-8")
    expect(icons).toHaveLength(1)
    expect(refreshButton.querySelector("[data-slot='button-loading-indicator']")).toBeNull()
    expect(refreshButton.querySelector("[data-slot='refresh-spinner']")).toHaveClass("loading-spinner")
  })

  it("keeps fast refresh feedback visible for the minimum duration", () => {
    vi.useFakeTimers()
    const onRefresh = vi.fn()
    const props = {
      selectedCount: 0,
      onRefresh,
      refreshLabel: "刷新",
      showAddButton: false,
      showBulkAdd: false,
    }
    const { rerender } = render(<TableActions {...props} isRefreshing={false} />)
    const refreshButton = screen.getByRole("button", { name: "刷新" })

    fireEvent.click(refreshButton)
    expect(onRefresh).toHaveBeenCalledTimes(1)
    expect(refreshButton).toHaveAttribute("aria-busy", "true")
    expect(refreshButton.querySelector("[data-slot='refresh-spinner']")).not.toBeNull()

    rerender(<TableActions {...props} isRefreshing={false} />)
    expect(refreshButton).toHaveAttribute("aria-busy", "true")

    act(() => {
      vi.advanceTimersByTime(299)
    })
    expect(refreshButton.querySelector("[data-slot='refresh-spinner']")).not.toBeNull()

    act(() => {
      vi.advanceTimersByTime(1)
    })
    expect(refreshButton).toHaveAttribute("aria-busy", "false")
    expect(refreshButton.querySelector("[data-slot='refresh-spinner']")).toBeNull()
  })

  it("keeps refresh feedback visible while the request is still pending", () => {
    vi.useFakeTimers()
    render(
      <TableActions
        selectedCount={0}
        onRefresh={vi.fn()}
        refreshLabel="刷新"
        isRefreshing
        showAddButton={false}
        showBulkAdd={false}
      />
    )

    const refreshButton = screen.getByRole("button", { name: "刷新" })
    act(() => {
      vi.advanceTimersByTime(1000)
    })

    expect(refreshButton).toHaveAttribute("aria-busy", "true")
    expect(refreshButton.querySelector("[data-slot='refresh-spinner']")).not.toBeNull()
  })
})
