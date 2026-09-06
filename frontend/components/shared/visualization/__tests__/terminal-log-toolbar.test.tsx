import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { useState } from "react"
import { describe, expect, it, vi } from "vitest"

import { TerminalLogToolbar } from "../terminal-log-toolbar"

const LOG_WINDOW_OPTIONS = [100, 200, 500, 1000, 2000, 5000] as const

function ToolbarProbe({ loading = false, onWindowChange = vi.fn() }: {
  loading?: boolean
  onWindowChange?: (value: number) => void
}) {
  const [windowSize, setWindowSize] = useState<number>(100)

  return (
    <TerminalLogToolbar
      searchTerm=""
      levelFilter="all"
      levelLabels={{
        all: "All",
        error: "Error",
        warn: "Warn",
        info: "Info",
        debug: "Debug",
      }}
      lineWindow={{
        value: windowSize,
        options: LOG_WINDOW_OPTIONS,
        onValueChange: (value) => {
          setWindowSize(value)
          onWindowChange(value)
        },
        label: "Log window",
      }}
      loading={loading}
    />
  )
}

describe("TerminalLogToolbar line window interactions", () => {
  it("renders exactly the six supported windows and updates the controlled value", async () => {
    const onWindowChange = vi.fn()

    render(<ToolbarProbe onWindowChange={onWindowChange} />)

    const trigger = screen.getByRole("combobox", { name: "Log window" })
    expect(trigger).toHaveTextContent("100")

    fireEvent.click(trigger)

    expect((await screen.findAllByRole("option")).map((option) => option.textContent?.trim())).toEqual(
      LOG_WINDOW_OPTIONS.map(String),
    )

    const option = screen.getByRole("option", { name: "200" })
    fireEvent.mouseMove(option)
    fireEvent.click(option)

    await waitFor(() => {
      expect(trigger).toHaveTextContent("200")
    })
    expect(onWindowChange).toHaveBeenLastCalledWith(200)
  })

  it("selects a window from the keyboard", async () => {
    const onWindowChange = vi.fn()

    render(<ToolbarProbe onWindowChange={onWindowChange} />)

    const trigger = screen.getByRole("combobox", { name: "Log window" })
    trigger.focus()
    fireEvent.keyDown(trigger, { key: "ArrowDown" })

    expect(await screen.findAllByRole("option")).toHaveLength(LOG_WINDOW_OPTIONS.length)

    const option = screen.getByRole("option", { name: "200" })
    option.focus()
    fireEvent.keyDown(option, { key: "Enter" })

    await waitFor(() => {
      expect(trigger).toHaveTextContent("200")
    })
    expect(onWindowChange).toHaveBeenLastCalledWith(200)
  })

  it("disables the selector while the toolbar is loading", () => {
    render(<ToolbarProbe loading />)

    expect(screen.getByRole("combobox", { name: "Log window" })).toBeDisabled()
  })

  it("keeps search usable and moves the level controls to a full mobile row", () => {
    const { container } = render(<ToolbarProbe />)

    const toolbar = container.querySelector("[data-slot='terminal-log-toolbar']")
    const searchShell = screen.getByRole("searchbox").closest("div")
    const levelFilters = container.querySelector("[data-slot='terminal-log-level-filters']")

    expect(toolbar).toHaveClass("flex-wrap", "min-h-10", "sm:flex-nowrap")
    expect(searchShell).toHaveClass("min-w-28", "sm:min-w-0")
    expect(levelFilters).toHaveClass("w-full", "sm:w-auto")
  })
})
