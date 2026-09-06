import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { RawLogViewer } from "../raw-log-viewer"

describe("RawLogViewer", () => {
  it("colorizes second-precision scan log timestamps and levels", () => {
    const { container } = render(
      <RawLogViewer content="[2026-07-24 20:09:32] [INFO] stage started" />
    )
    const viewer = container.querySelector('[data-slot="raw-log-viewer"]')

    expect(viewer?.textContent).toBe("[2026-07-24 20:09:32] [INFO] stage started")
    expect(viewer?.querySelectorAll("span")).toHaveLength(2)
  })

  it("renders HTML-like log content as inert text", () => {
    const payload = '<img src=x onerror="window.__logExecuted = true">'
    const { container } = render(<RawLogViewer content={payload} />)
    const viewer = container.querySelector('[data-slot="raw-log-viewer"]')

    expect(viewer).not.toBeNull()
    expect(viewer?.querySelector("img")).toBeNull()
    expect(viewer?.textContent).toBe(payload)
  })

  it("preserves ANSI rendering while escaping embedded markup", () => {
    const { container } = render(
      <RawLogViewer content={'\u001b[31mERROR <script>unsafe()</script>\u001b[0m'} />
    )
    const viewer = container.querySelector('[data-slot="raw-log-viewer"]')

    expect(viewer?.querySelector("script")).toBeNull()
    expect(viewer?.querySelector("span")).not.toBeNull()
    expect(viewer?.textContent).toContain("ERROR <script>unsafe()</script>")
  })

  it("does not force a reader back to the bottom after they scroll up", () => {
    const { container, rerender } = render(<RawLogViewer content="first" />)
    const viewer = container.querySelector('[data-slot="raw-log-viewer"]') as HTMLPreElement

    Object.defineProperties(viewer, {
      clientHeight: { configurable: true, value: 100 },
      scrollHeight: { configurable: true, value: 500 },
    })
    viewer.scrollTop = 100
    fireEvent.scroll(viewer)

    rerender(<RawLogViewer content={'first\nsecond'} />)

    expect(viewer.scrollTop).toBe(100)
  })

  it("keeps an optional copy action in the terminal content top-right corner", () => {
    const { container } = render(
      <RawLogViewer
        content="scan started"
        topRightAction={<button type="button">复制全部日志</button>}
      />,
    )

    const actionSlot = container.querySelector('[data-slot="raw-log-top-right-action"]')
    const button = screen.getByRole("button", { name: "复制全部日志" })

    expect(actionSlot?.contains(button)).toBe(true)
  })
})
